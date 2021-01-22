package rabbitmq

import (
	"BakatoraPushServer/src/util"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"sync"
	"time"
)

type Queue struct {
	mq      *RabbitMq
	Name    string
	iq      *amqp.Queue //internal queue object
	channel *amqp.Channel
	logger  *log.Logger
}

type Exchange struct {
	Name string
}

type RabbitMq struct {
	sync.RWMutex
	connId     int
	user       string
	pwd        string
	host       string
	connection *amqp.Connection
	queues     map[string]*Queue
	exchanges  map[string]*Exchange
	logger     *log.Logger
	logfile    string
	channel    *amqp.Channel
}

func New(connId int, user, pwd, host, logfile string) *RabbitMq {
	mq := new(RabbitMq)
	mq.connId = connId
	mq.user = user
	mq.pwd = pwd
	mq.host = host
	mq.logger = util.CreateLogger(" [RabbitMq] ", logfile)
	mq.logfile = logfile
	err := mq.connect()
	if err != nil {
		mq.logger.Fatalf("Initialize conncection failed, exit.")
	}
	mq.queues = make(map[string]*Queue)
	mq.exchanges = make(map[string]*Exchange)
	return mq
}

func (mq *RabbitMq) connect() error {
	RabbitUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/", mq.user, mq.pwd, mq.host, mq.connId)
	mqConn, err := amqp.Dial(RabbitUrl)
	if err != nil {
		mq.logger.Printf("Dial failed url:%s, err: %v.\n", RabbitUrl, err)
		return err
	}
	mq.connection = mqConn
	errCh := make(chan *amqp.Error)
	mq.connection.NotifyClose(errCh)
	go func() {
		err := <-errCh
		mq.logger.Printf("connection close for reason: %s", err.Reason)
		mq.reconnect()
	}()
	mq.channel, err = mq.GetChannel()
	if err != nil {
		mq.logger.Printf("Get global channel failed: %v", err)
		return err
	}
	return nil
}

func (mq *RabbitMq) reconnect() {
	count := 0
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			count++
			mq.logger.Printf("reconnect times %d", count)
			err := mq.connect()
			if err == nil {
				return
			}
		}
	}

}

func (mq *RabbitMq) CreateQueue(queueName, key, exchange string, durable, noWait bool) (q *Queue, err error) {
	//create and bind to exchange
	queue := mq.queues[queueName]
	if queue != nil {
		return queue, nil
	}
	channel, err := mq.GetChannel()
	if err != nil {
		mq.logger.Printf("Get channel failed: %v", err)
		return nil, err
	}
	defer channel.Close()
	iq, err := channel.QueueDeclarePassive(queueName, true, false, false, noWait, nil)
	if err != nil {
		channel, _ = mq.GetChannel()
		iq, err = channel.QueueDeclare(queueName, durable, false, false, noWait, nil)
		if err != nil {
			mq.logger.Printf("Open queue name %s failed: %s", queueName, err)
			return nil, err
		}
	}
	q = new(Queue)
	q.init(queueName, mq, &iq, mq.logfile)
	err = channel.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		mq.logger.Printf("Bind queueName failed.queueName: %s, key: %s, exchange: %s, err: %s", queueName, key, exchange, err)
		return nil, err
	}
	mq.queues[q.Name] = q
	return q, err
}

func (mq *RabbitMq) CreateExchange(exchangeName, exchangeType string, durable, noWait bool) (exchange *Exchange, err error) {
	exchange = mq.exchanges[exchangeName]
	if exchange != nil {
		return exchange, nil
	}
	channel, err := mq.GetChannel()
	if err != nil {
		mq.logger.Printf("Get channel failed: %v", err)
		return nil, err
	}
	defer channel.Close()
	//先检查是否exchange是否已经存在
	err = channel.ExchangeDeclarePassive(exchangeName, exchangeType, durable, false, false, noWait, nil)
	if err != nil {
		channel, _ = mq.GetChannel()
		err = channel.ExchangeDeclare(exchangeName, exchangeType, durable, false, false, noWait, nil)
		if err != nil {
			mq.logger.Printf("create exchange err: %v", err)
			return nil, err
		}
	}
	exchange = new(Exchange)
	exchange.Name = exchangeName
	mq.exchanges[exchangeName] = exchange
	return exchange, err
}

func (mq *RabbitMq) Publish(exchangeName, routingKey, msg, expiration string) (err error) {
	err = mq.channel.Publish(exchangeName, routingKey, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(msg),
		Expiration:  expiration,
	})
	if err != nil {
		mq.logger.Printf("MQ任务发送失败:%s \n", err)
	}
	return err
}

func (mq *RabbitMq) GetChannel() (*amqp.Channel, error) {
	return mq.connection.Channel()
}

func (q *Queue) Consume() (msgCh <-chan amqp.Delivery, err error) {
	q.channel, err = q.mq.GetChannel()
	if err != nil {
		q.logger.Printf("open channel failed: %v", err)
		return nil, err
	}
	err = q.channel.Qos(1, 0, true)
	if err != nil {
		q.logger.Printf("Qos failed:%s \n", err)
		return nil, err
	}
	return q.channel.Consume(q.Name, q.Name, false, false, false, false, nil)
}

func (q *Queue) Cancel() {
	go func() {
		var err error
		if q.channel != nil {
			q.channel, err = q.mq.GetChannel()
			if err != nil {
				q.logger.Printf("cancel when get channel failed: %v", q.channel)
				return
			}
		}
		_, err = q.channel.QueueDelete(q.Name, false, false, false)
		for err != nil {
			_, err = q.channel.QueueDelete(q.Name, false, false, false)
		}
	}()
}

func (q *Queue) Stop() {
	//q.StopCh <- struct{}{}
	q.channel.Close()
}

func (q *Queue) init(queueName string, mq *RabbitMq, iq *amqp.Queue, logfile string) {
	q.Name = queueName
	q.mq = mq
	q.iq = iq
	loggerPrefix := fmt.Sprintf(" [Queue:%s] ", q.Name)
	q.logger = util.CreateLogger(loggerPrefix, logfile)
}
