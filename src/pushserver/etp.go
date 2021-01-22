package pushserver

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/format"
	"github.com/go-netty/go-netty/codec/frame"
	"github.com/go-netty/go-netty/transport/tcp"
	"log"
)

const (
	Alarm             EventType = 1
	Monitor           EventType = 2
	Hello             MsgType   = 1
	Bye               MsgType   = 2
	LengthFieldLength           = 4
)

type (
	EventType int

	EtpConfig struct {
		port   int
		server *PushServer
	}

	EventTransportServer struct {
		config    EtpConfig
		bootstrap *netty.Bootstrap
	}

	EventHandler struct {
		server *PushServer
	}

	MsgType int

	EtpClientMsg struct {
		ClientId string  `json:"clientId"`
		MsgType  MsgType `json:"msgType,int"`
	}

	EtpClientRes struct {
		Success bool `json:"success,bool"`
	}
)

func (etps *EventTransportServer) Start() {
	var bootstrap = netty.NewBootstrap()
	etps.bootstrap = &bootstrap
	bootstrap.ChildInitializer(func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(frame.LengthFieldCodec(binary.LittleEndian, 1024, 0, LengthFieldLength, 0, 4)).
			AddLast(format.TextCodec()).
			AddLast(EventHandler{etps.config.server})
	})
	url := fmt.Sprintf("tcp://0.0.0.0:%d", etps.config.port)
	bootstrap.
		Transport(tcp.New()).
		Listen(url)
}

func NewEtpServer(config EtpConfig) *EventTransportServer {
	etps := new(EventTransportServer)
	etps.config = config
	return etps
}

func (eh EventHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
	msg, _ := message.(string)
	echm := EtpClientMsg{}
	err := json.Unmarshal([]byte(msg), &echm)
	if err != nil {
		log.Printf("invalid args: err: %v", err)
	}
	if echm.MsgType == Hello {
		consumer := eh.server.ClientConn(echm.ClientId, ctx)
		ctx.SetAttachment(consumer)
		echr := EtpClientRes{}
		echr.Success = true
		data, _ := json.Marshal(echr)
		ctx.Write(string(data))
	} else if echm.MsgType == Bye {
		consumer, ok := ctx.Attachment().(*Consumer)
		if ok {
			log.Printf("%s conn disconnect", consumer.ClientId)
			ctx.SetAttachment(nil)
			consumer.disconnect()
			ctx.Close(nil)
			//return
		}
	}
	//ctx.HandleRead(msg)
}

func (eh EventHandler) HandleException(ctx netty.ExceptionContext, ex netty.Exception) {
	// 向后续的handler传递控制权
	// 如果是最后一个handler或者需要中断请求可以不用调用
	log.Printf("conn exception: %v", ex)
	consumer, ok := ctx.Attachment().(*Consumer)
	if ok {
		consumer.disconnect()
	}
	ctx.Close(ex)
	ctx.HandleException(ex)
}
