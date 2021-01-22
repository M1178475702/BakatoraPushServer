package pushserver

import (
	"BakatoraPushServer/src/pushserver/db"
	"BakatoraPushServer/src/rabbitmq"
	"BakatoraPushServer/src/util"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-netty/go-netty"
	"log"
	"strconv"
	"sync"
)

type Producer struct {
	ServerName string
	Exchange   *rabbitmq.Exchange
	Desc       string
	Events     map[string]*Event
}

type PushServer struct {
	prodRwMu  sync.RWMutex
	consRwMu  sync.RWMutex
	cfg       Config
	mq        *rabbitmq.RabbitMq
	prods     map[string]*Producer
	consumers map[string]*Consumer
	logger    *log.Logger
	etps      *EventTransportServer
	ge        *gin.Engine
	db        *db.DB
}

/************************************** public *********************************************/
/*
	1. 连接amqp
	2. 监听http，设置处理程序
	3. 监听socket，设置处理程序
*/
func (ps *PushServer) Start(config Config) {
	ps.cfg = config
	ps.logger = util.CreateLogger("[PushServer] ", ps.cfg.Logfile)
	ps.db = db.New(ps.cfg.DbUser, ps.cfg.DbPwd, ps.cfg.DbHost, ps.cfg.DbPort, ps.cfg.DbName)
	ps.mq = rabbitmq.New(ps.cfg.MqConnId, ps.cfg.MqUser, ps.cfg.MqPwd, ps.cfg.MqHost, ps.cfg.Logfile)
	etpConfig := EtpConfig{port: ps.cfg.EtpListenPort, server: ps}
	ps.etps = NewEtpServer(etpConfig)
	ps.etps.Start()
	ps.prods = map[string]*Producer{}
	ps.consumers = map[string]*Consumer{}
	ps.ge = gin.Default()
	SetupInnerEngine(ps)
	ps.ge.Run(":" + strconv.Itoa(ps.cfg.HttpListenPort))
}

func (ps *PushServer) RegisterProducer(args *RegisterServerArgs) (res *RegisterServerRes) {
	res = new(RegisterServerRes)
	serverName := args.ServerName
	var err error
	ps.prodRwMu.Lock()
	defer ps.prodRwMu.Unlock()
	producer := ps.getProducer(serverName)
	if producer == nil {
		producer, err = ps.createProducer(serverName, args.Desc)
		if err != nil {
			ps.logger.Printf("Create producer failed. ServerName: %s,err: %s", serverName, err)
			res.Success = false
			res.Msg = err.Error()
			return res
		}
	}
	ps.setProducer(producer)
	res.Success = true
	return res
}

func (ps *PushServer) RegisterEvent(args *RegisterEventArgs) (res *RegisterEventRes) {
	res = new(RegisterEventRes)
	ps.prodRwMu.Lock()
	defer ps.prodRwMu.Unlock()
	event, err := ps.getEvent(args.ServerName, args.EventName)
	if err != nil {
		res.Success = false
		res.Msg = err.Error()
		return res
	}
	if event == nil {
		event, err = ps.createEvent(args.ServerName, args.EventName, args.EventDesc, args.EventType, args.Fields)
		if err != nil {
			ps.logger.Printf("create event failed.ServerName EventName: %s,event EventName: %s,err: %s", args.ServerName, args.EventName, err)
			res.Success = false
			res.Msg = err.Error()
			return res
		}
		ps.getProducer(args.ServerName).Events[args.EventName] = event
	}
	res.Success = true
	res.RouterKey = event.q.Name
	return res
}

func (ps *PushServer) PushEvent(args *PushEventArgs) (res * PushEventRes){
	res = new(PushEventRes)
	producer := ps.getProducer(args.ServerName)
	if producer == nil {
		res.Success = false
		res.Msg = fmt.Sprintf("Get producer %s failed: producer not exist", args.ServerName)
		return
	}
	event, err := ps.getEvent(args.ServerName, args.EventName)
	if err != nil {
		res.Success = false
		res.Msg = fmt.Sprintf("Get event %s.%s failed: %v", args.ServerName, args.EventName, err)
		return
	}
	entry, err := json.Marshal(args.Entry)
	if err != nil {
		res.Success = false
		res.Msg = fmt.Sprintf("Invalid entry %s.%s failed: %v", args.ServerName, args.EventName, err)
		return
	}
	err = ps.mq.Publish(producer.Exchange.Name, event.q.Name, string(entry), args.TTL)
	if err != nil {
		res.Success = false
		res.Msg = fmt.Sprintf("Publish event %s.%s failed: %v",  args.ServerName, args.EventName, err)
		return
	}
	res.Success = true
	return res
}

func (ps *PushServer) CancelEvent(args *CancelEventArgs) (res *CancelEventRes) {
	res = new(CancelEventRes)
	event, err := ps.getEvent(args.ServerName, args.EventName)
	if err != nil {
		res.Success = false
		res.Msg = err.Error()
		return
	}
	ps.deleteEvent(event)
	event.Cancel()
	res.Success = true
	return res
}

func (ps *PushServer) Subscribe(args *SubscribeArgs) *SubscribeRes {
	res := new(SubscribeRes)
	ps.prodRwMu.RLock()
	event, err := ps.getEvent(args.ServerName, args.EventName)
	ps.prodRwMu.RUnlock()
	if err != nil {
		res.Success = false
		res.Msg = fmt.Sprintf("event %s.%s not exist", args.ServerName, args.ServerName)
		return res
	}
	ps.consRwMu.Lock()
	defer ps.consRwMu.Unlock()
	consumer := ps.getConsumer(args.ClientId)
	if consumer == nil {
		res.Success = false
		res.Msg = fmt.Sprintf("client id %s not exist", args.ClientId)
		return res
	}
	consumer.AddEvent(event)
	event.AddConsumer(consumer)
	res.Success = true
	return res
}

func (ps *PushServer) Unsubscribe(args *UnsubscribeArgs) (res *UnsubscribeRes) {
	res = new(UnsubscribeRes)
	consumer := ps.getConsumer(args.ClientId)
	if consumer == nil {
		res.Success = false
		res.Msg = fmt.Sprintf("client id %s not exist", args.ClientId)
		return res
	}
	event, err := ps.getEvent(args.ServerName, args.EventName)
	if err != nil {
		res.Success = false
		res.Msg = fmt.Sprintf("event %s.%s not exist", args.ServerName, args.ServerName)
		return res
	}
	event.DeleteConsumer(consumer)
	res.Success = true
	return res

}

func (ps *PushServer) GetServerList(args *GetServerListArgs) (res *GetServerListRes) {
	ps.prodRwMu.RLock()
	defer ps.prodRwMu.RUnlock()
	i := 0
	res = new(GetServerListRes)
	lenMap := len(ps.prods)
	res.List = make([]ServerListEle, lenMap)
	for k, v := range ps.prods {
		res.List[i] = ServerListEle{
			ServerName: k,
			ServerDesc: v.Desc,
		}
	}
	res.Count = lenMap
	res.Success = true
	return res
}

func (ps *PushServer) GetEventList(args *GetEventListArgs) (res *GetEventListRes) {
	ps.prodRwMu.RLock()
	defer ps.prodRwMu.RUnlock()
	res = new(GetEventListRes)
	producer := ps.getProducer(args.ServerName)
	if producer == nil {
		res.Msg = fmt.Sprintf("server %s not exist.", args.ServerName)
		return res
	}
	lenMap := len(producer.Events)
	res.List = make([]EventListEle, lenMap)
	i := 0
	for k, v := range producer.Events {
		res.List[i] = EventListEle{
			EventName:  k,
			ServerName: args.ServerName,
			EventDesc:  v.Desc,
			EventType:  v.Type,
			Fields:     v.Fields,
		}
		i++
	}
	res.Count = lenMap
	res.Success = true
	return res
}

func (ps *PushServer) ClientLogIn(args *ClientLogInArgs) (res *ClientLogInRes) {
	res = new(ClientLogInRes)
	client, err := ps.db.Client.FindByClientName(args.ClientName)
	if err != nil {
		ps.logger.Printf("find client by name %s faile: %v", args.ClientName, err)
		res.Msg = err.Error()
		res.Success = false
		return
	}
	if args.ClientName == client.ClientName && args.Password == client.Password {
		res.ClientId = client.ClientId
		res.Success = true
	} else {
		res.Msg = "账号不存在或者密码错误"
		res.Success = false
	}
	return res
}

func (ps *PushServer) ClientConn(clientId string, ctx netty.InboundContext) *Consumer {
	ps.consRwMu.Lock()
	defer ps.consRwMu.Unlock()
	consumer := ps.getConsumer(clientId)
	if consumer == nil {
		consumer = ps.createConsumer(clientId, ctx)
		ps.consumers[clientId] = consumer
	} else {
		consumer.Conn = ctx
	}
	return consumer
}

func (ps *PushServer) ClientDisconnect(clientId string) {
	ps.consRwMu.Lock()
	defer ps.consRwMu.Unlock()
	delete(ps.consumers, clientId)
}

/************************************** private *********************************************/
func (ps *PushServer) createProducer(serverName, desc string) (producer *Producer, err error) {
	exchangeName := ps.generateExchangeName(serverName)
	ex, err := ps.mq.CreateExchange(exchangeName, "direct", false, false)
	if err != nil {
		return nil, err
	}
	producer = new(Producer)
	producer.Exchange = ex
	producer.Events = map[string]*Event{}
	producer.ServerName = serverName
	producer.Desc = desc
	return producer, nil
}

func (ps *PushServer) generateExchangeName(serverName string) string {
	return fmt.Sprintf("producer.%s", serverName)
}

func (ps *PushServer) getProducer(serverName string) *Producer {
	return ps.prods[serverName]
}

func (ps *PushServer) setProducer(producer *Producer) {
	ps.prods[producer.ServerName] = producer
}

func (ps *PushServer) getEvent(serverName, eventName string) (*Event, error) {
	producer := ps.getProducer(serverName)
	if producer == nil {
		return nil, errors.New(fmt.Sprintf("Server %s do not exist\n", serverName))
	}
	return producer.Events[eventName], nil
}

func (ps *PushServer) createEvent(serverName, eventName, desc string, eventType EventType, fields []Field) (event *Event, err error) {
	interQueueName := ps.generateInterQueueName(serverName, eventName)
	exchangeName := ps.getProducer(serverName).Exchange.Name
	//also need create queue
	q, err := ps.mq.CreateQueue(interQueueName, interQueueName, exchangeName, true, false)
	if err != nil {
		return nil, err
	} else {
		event = new(Event)
		event.init(serverName, eventName, desc, eventType, fields, q, ps.cfg.Logfile)
		return event, err
	}
}

func (ps *PushServer) deleteEvent(event *Event) {
	events := &(ps.getProducer(event.ServerName).Events)
	delete(*events, event.EventName)
}

func (ps *PushServer) generateInterQueueName(serverName, eventName string) string {
	return fmt.Sprintf("event.%s.%s", serverName, eventName)
}

func (ps *PushServer) getConsumer(clientId string) *Consumer {
	return ps.consumers[clientId]
}

func (ps *PushServer) createConsumer(clientId string, ctx netty.InboundContext) *Consumer {
	consumer := new(Consumer)
	consumer.ClientId = clientId
	consumer.Conn = ctx
	consumer.SubscribeEvents = make([]*Event, 0, 3)
	consumer.Enable = true
	consumer.ps = ps
	return consumer
}
