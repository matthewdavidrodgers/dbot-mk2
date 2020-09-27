package main

import (
	"github.com/matthewdavidrodgers/dbot-mk2/dbot"
	"github.com/matthewdavidrodgers/dbot-mk2/mcserver"
)

func main() {
	serverRequests := make(chan *mcserver.ServerRequestOp)
	discordResponses := make(chan string)

	mcserver.MakeServerManager(serverRequests, discordResponses)
	dbot.MakeBotManager(serverRequests, discordResponses)
}
