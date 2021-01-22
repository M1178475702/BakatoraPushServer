package pushserver

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
)

func parseBody(ctx *gin.Context, pointer interface{}) (err error){
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, pointer)
	if err != nil {
		return err
	}
	return nil
}


func SetupInnerEngine(ps *PushServer){
	ge := ps.ge
	{
		serverSide := ge.Group("/server-side")

		serverSide.POST("/register/server", func(ctx *gin.Context) {
			args := new(RegisterServerArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.RegisterProducer(args)
			ctx.JSON(200, reply)
		})

		serverSide.POST("/register/event", func(ctx *gin.Context) {
			args := new(RegisterEventArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.RegisterEvent(args)
			ctx.JSON(200, reply)
		})

		serverSide.POST("/event/cancel", func(ctx *gin.Context) {
			args := new(CancelEventArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.CancelEvent(args)
			ctx.JSON(200, reply)
		})

		serverSide.POST("/publish", func(ctx *gin.Context) {
			args := new(PushEventArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.PushEvent(args)
			ctx.JSON(200, reply)
		})
	}

	{
		clientSide := ge.Group("/client-side")

		clientSide.POST("/login", func(ctx *gin.Context) {
			args := new(ClientLogInArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.ClientLogIn(args)
			ctx.JSON(200, reply)
		})

		clientSide.POST("/subscribe", func(ctx *gin.Context) {
			args := new(SubscribeArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.Subscribe(args)
			ctx.JSON(200, reply)
		})

		clientSide.GET("/server/list", func(ctx *gin.Context) {
			args := GetServerListArgs{
				ClientId:   ctx.Query("clientId"),
			}
			reply := ps.GetServerList(&args)
			ctx.JSON(200, reply)
		})

		clientSide.GET("/event/list", func(ctx *gin.Context) {
			args := GetEventListArgs{
				ClientId:   ctx.Query("clientId"),
				ServerName: ctx.Query("serverName"),
			}
			reply := ps.GetEventList(&args)
			ctx.JSON(200, reply)
		})

		clientSide.POST("/unsubscribe", func(ctx *gin.Context){
			args := new(UnsubscribeArgs)
			err := parseBody(ctx, args)
			if err != nil {
				ps.logger.Printf("Invalid request body, err %v", err)
				return
			}
			reply := ps.Unsubscribe(args)
			ctx.JSON(200, reply)
		})
	}


}

