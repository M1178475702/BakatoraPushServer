package pushserver

import (
	"time"
)

type RegisterServerArgs struct {
	ServerName string `json:"serverName"`
	Desc       string `json:"desc"`
}

type RegisterServerRes struct {
	Success  bool   `json:"success,bool"`
	Msg      string `json:"msg"`
}

type RegisterEventArgs struct {
	ServerName string    `json:"serverName"`
	EventName  string    `json:"eventName"`
	EventType  EventType `json:"eventType,int"`
	EventDesc  string    `json:"eventDesc"`
	Fields     []Field   `json:"fields"`
}

type RegisterEventRes struct {
	RouterKey string `json:"routerKey"`
	Success   bool   `json:"success,bool"`
	Msg       string `json:"msg"`
}

type SubscribeArgs struct {
	ServerName string `json:"serverName"`
	EventName  string `json:"eventName"`
	ClientId   string `json:"clientId"`
}

type SubscribeRes struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type UnsubscribeArgs struct {
	ServerName string `json:"serverName"`
	EventName  string `json:"eventName"`
	ClientId   string `json:"clientId"`
}

type UnsubscribeRes struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type ClientLogInArgs struct {
	ClientName string `json:"clientName"`
	Password   string `json:"password"`
}

type ClientLogInRes struct {
	ClientId string `json:"clientId"`
	Success  bool   `json:"success"`
	Msg      string `json:"msg"`
}

type GetServerListArgs struct {
	ClientId string `json:"clientId"`
}

type GetServerListRes struct {
	List    []ServerListEle `json:"list"`
	Count   int             `json:"count"`
	Msg     string          `json:"msg"`
	Success bool            `json:"success"`
}

type ServerListEle struct {
	ServerName string `json:"serverName"`
	ServerDesc string `json:"serverDesc"`
}

type GetEventListArgs struct {
	ClientId   string `json:"clientId"`
	ServerName string `json:"serverName"`
}

type GetEventListRes struct {
	List    []EventListEle `json:"list"`
	Count   int            `json:"count,int"`
	Msg     string         `json:"msg"`
	Success bool           `json:"success"`
}
type Field struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type CancelEventArgs struct {
	ServerName string `json:"serverName"`
	EventName  string `json:"eventName"`
}

type CancelEventRes struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type EventListEle struct {
	EventName  string    `json:"eventName"`
	ServerName string    `json:"serverName"`
	EventDesc  string    `json:"eventDesc"`
	EventType  EventType `json:"eventType,int"`
	Fields     []Field   `json:"fields"`
}

type PushEventArgs struct {
	ServerName string           `json:"serverName"`
	EventName  string           `json:"eventName"`
	EventType  EventType        `json:"eventType,int"`
	Entry      SourceEventEntry `json:"entry"`
	TTL        string           `json:"ttl"`
}

type BaseHttpResult struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type PushEventRes struct {
	BaseHttpResult
}

//used for server side
type SourceEventEntry struct {
	Data     string    `json:"data"`
	PushTime time.Time `json:"pushTime"`
}

//used for user side
type EventEntry struct {
	EventName  string    `json:"eventName"`
	ServerName string    `json:"serverName"`
	EventType  EventType `json:"eventType,int"`
	Data       string    `json:"data"`
	PushTime   time.Time `json:"pushTime"`
	End        bool      `json:"end,bool"`
}
