package pushserver

import (
	"encoding/json"
	"github.com/go-netty/go-netty"
	"sync"
)

type Consumer struct {
	sync.Mutex
	ClientId        string
	ps              *PushServer
	Conn            netty.InboundContext
	SubscribeEvents []*Event
	Enable          bool
}

type ConsumerGroup struct {
	sync.Mutex
	Event *Event
	mem   []*Consumer
}

func (c *Consumer) notify(entry EventEntry) {
	data, _ := json.Marshal(entry)
	c.Conn.Write(string(data))
}

func (c *Consumer) disconnect() {

	c.Lock()          //c lock
	if c.Enable {
		c.Enable = false
		//不再调用Close，会引发cpu异常占用，由客户端主动关闭

		c.Conn = nil
		c.ps.ClientDisconnect(c.ClientId)

		for _, e := range c.SubscribeEvents {
			e.cg.DeleteConsumer(c.ClientId)  //cg lock
		}

		c.SubscribeEvents = nil

	}

	c.Unlock()
	//c.Conn.Close(nil)

}

func (c *Consumer) AddEvent(event *Event){
	c.Lock()
	c.SubscribeEvents = append(c.SubscribeEvents, event)
	c.Unlock()
}

func (c *Consumer) DeleteEvent(event *Event){
	c.Lock()
	for i, e := range c.SubscribeEvents {
		if e.q.Name == event.q.Name {
			c.SubscribeEvents = append(c.SubscribeEvents[:i], c.SubscribeEvents[i + 1:]...)
			break
		}
	}
	c.Unlock()
}

func (cg *ConsumerGroup) NotifyAll(entry EventEntry) {
	cg.Lock()
	defer cg.Unlock()
	for _, c := range cg.mem {
		c.notify(entry)
	}
}

func (cg *ConsumerGroup) isExist(consumer *Consumer) (int, bool) {
	for i, c := range cg.mem {
		if c.ClientId == consumer.ClientId {
			return i, true
		}
	}
	return -1, false
}

func (cg *ConsumerGroup) Add(consumer *Consumer) {
	cg.Lock()
	defer cg.Unlock()
	if i, isExist := cg.isExist(consumer); isExist {
		cg.deleteConsumer(i)
		cg.mem = append(cg.mem, consumer)
	} else {
		cg.mem = append(cg.mem, consumer)
	}
}

func (cg *ConsumerGroup) DeleteConsumer(clientId string) {
	cg.Lock()
	defer cg.Unlock()
	if len(cg.mem) <= 0 {
		return
	}
	for i, c := range cg.mem {
		if c.ClientId == clientId {
			cg.deleteConsumer(i)
			break
		}
	}
	if len(cg.mem) <= 0 && cg.Event.recving {
		cg.Event.Stop()
	}
}

func (cg *ConsumerGroup) DeleteAll(){
	cg.mem = nil
}

func (cg *ConsumerGroup) deleteConsumer(index int) {
	//不需要 disconnect
	cg.mem = append(cg.mem[:index], cg.mem[index+1:]...)
}
