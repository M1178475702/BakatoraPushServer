package pushserver

type Config struct {
	HttpListenPort int
	EtpListenPort  int
	MqConnId       int
	MqHost         string
	MqUser         string
	MqPwd          string
	Logfile        string
	DbUser         string
	DbPwd          string
	DbHost         string
	DbPort         string
	DbName         string
}
