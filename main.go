package main

import (
	"github.com/matthewdavidrodgers/dbot-mk2/dbot"
	"github.com/matthewdavidrodgers/dbot-mk2/defs"
	"github.com/matthewdavidrodgers/dbot-mk2/mcserver"
)

func main() {
	serverRequests := make(chan *defs.ServerRequestOp)
	discordResponses := make(chan string)

	mcserver.MakeServerManager(serverRequests, discordResponses)
	dbot.MakeBotManager(serverRequests, discordResponses)
}
