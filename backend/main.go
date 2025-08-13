package main

import (
	"github.com/mudit06mah/CloudIde/aws"
	"github.com/mudit06mah/CloudIde/ws"
	"github.com/mudit06mah/CloudIde/config"
)

func main() {
	config.LoadEnv()
	aws.InitAWSConfig()
	ws.StartWebSocketServer()
}	