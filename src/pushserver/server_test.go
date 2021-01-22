package pushserver

import (
	"log"
	"testing"
)

func TestPushServer_Start(t *testing.T) {
	ps := new(PushServer)
	config := Config{
		MqConnId:       5672,
		HttpListenPort: 3401,
		EtpListenPort:  3402,
		Logfile:        "",
	}
	ps.Start(config)
	log.Println("push server start")
}
