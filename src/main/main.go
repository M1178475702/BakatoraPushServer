package main

import (
	"BakatoraPushServer/src/pushserver"
	"encoding/json"
	"log"
	"os"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("Config file path is required!")
	}
	configFilePath := args[1]
	if configFilePath != "test" {
		file, err := os.Open(configFilePath)
		if err != nil {
			log.Fatalf("Open file %s failed.", configFilePath)
		}
		dec := json.NewDecoder(file)
		config := new(pushserver.Config)
		err = dec.Decode(config)
		if err != nil {
			log.Fatalf("Invalid data for config in file %s,please give valid config!", configFilePath)
		}
		ps := new(pushserver.PushServer)
		ps.Start(*config)

	} else {
		ps := new(pushserver.PushServer)
		config := pushserver.Config{
			MqConnId:       5672,
			MqHost:         "localhost",
			MqUser:         "guest",
			MqPwd:          "guest",
			HttpListenPort: 3401,
			EtpListenPort:  3402,
			Logfile:        "",
			DbUser:         "root",
			DbPwd:          "123456",
			DbHost:         "localhost",
			DbPort:         "3306",
			DbName:         "push_server",
		}
		ps.Start(config)
	}
}
