package pushserver

import (
	"BakatoraPushServer/src/rabbitmq"
	"BakatoraPushServer/src/util"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type Event struct {
	sync.Mutex
	EventName  string
	ServerName string
	Desc       string
	Type       EventType
	Fields     []Field
	cg         *ConsumerGroup
	q          *rabbitmq.Queue
	recving    bool
	logger     *log.Logger
	stopCh     chan struct{}
}

func (e *Event) init(serverName, eventName, desc string, eventType EventType, fields []Field, q *rabbitmq.Queue, logfile string) {
	e.EventName = eventName
	e.ServerName = serverName
	e.Type = eventType
	e.Desc = desc
	e.Fields = fields
	e.q = q
	e.recving = false
	e.stopCh = make(chan struct{})
	cg := new(ConsumerGroup)
	cg.Event = e
	cg.mem = make([]*Consumer, 0, 5)
	e.cg = cg
	e.logger = util.CreateLogger(fmt.Sprintf("[Event %s] ", e.q.Name), logfile)
}

func (e *Event) AddConsumer(consumer *Consumer) {
	e.cg.Add(consumer)
	e.Lock()
	if !e.recving {
		err := e.beginRecv()
		if err != nil {
			e.logger.Printf("begin wait failed: %v", err)
			e.cg.DeleteConsumer(consumer.ClientId)
		} else {
			e.recving = true
		}
	}
	e.Unlock()
}

func (e *Event) DeleteConsumer(consumer *Consumer) {
	e.cg.DeleteConsumer(consumer.ClientId)
}

func (e *Event) beginRecv() (err error) {
	//等待队列
	msgCh, err := e.q.Consume()
	if err != nil {
		e.logger.Printf("get consumer channel failed: %v", err)
		return err
	}
	go func(event *Event) {
	mainLoop:
		for {
			select {
			case msg := <-msgCh:
				if msg.Body == nil {
					var count = 0
					for {
						select {
						case <-e.stopCh:
							return
						case <-time.After(time.Second):
							msgCh, err = e.q.Consume()
							if err != nil {
								e.logger.Printf("get consumer channel failed: %v.\n", err)
								count++
								e.logger.Printf("retry get channel for %d times..\n", count)
							} else {
								continue mainLoop
							}
						}
					}
				}
				body := string(msg.Body)
				srcEntry := new(SourceEventEntry)
				err := json.Unmarshal([]byte(body), srcEntry)
				if err != nil {
					e.logger.Printf("Invalid event entry for server name: %s, event name: %s", event.ServerName, event.EventName)
					//discard the wrong msg
					_ = msg.Ack(true)
					continue
				}
				entry := EventEntry{
					EventName:  event.EventName,
					ServerName: event.ServerName,
					Data:       srcEntry.Data,
					PushTime:   srcEntry.PushTime,
					EventType:  event.Type,
					End:        false,
				}
				event.cg.NotifyAll(entry)
				_ = msg.Ack(true)
			case <-e.stopCh:
				return
			}
		}
	}(e)
	return nil
}

func (e *Event) Cancel() {
	//结束consume 删除 mq的routerKey，queue，，像所有consumer发送end entry，删除所有consumer
	e.Stop()
	e.q.Cancel()
	endEntry := EventEntry{
		EventName:  e.EventName,
		ServerName: e.ServerName,
		EventType:  e.Type,
		Data:       "",
		PushTime:   time.Now(),
		End:        true,
	}
	e.cg.NotifyAll(endEntry)
	e.cg.DeleteAll()
}

func (e *Event) Stop() {
	e.Lock()
	e.recving = false
	e.q.Stop()
	e.stopCh <- struct{}{}
	e.Unlock()
}
