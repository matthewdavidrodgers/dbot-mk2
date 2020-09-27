package dbot

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/matthewdavidrodgers/dbot-mk2/mcserver"
)

// MakeBotManager starts discord bot that listens to incoming messages, and sends ServerRequestOps when a valid
// command is requested. it also sends messages back to the discord server based on the messages provided by the
// discordResponses channel
func MakeBotManager(serverRequests chan<- *mcserver.ServerRequestOp, discordResponses chan string) {
	bg := context.Background()
	client := disgord.New(disgord.Config{
		BotToken: os.Getenv("BOT_TOKEN"),
	})
	var channelID disgord.Snowflake
	fmt.Println(client)

	defer client.StayConnectedUntilInterrupted(bg)

	handleMessage := func(session disgord.Session, evt *disgord.MessageCreate) {
		msg := evt.Message
		if channelID == 0 {
			// NOTE: FLAKY
			// WILL ONLY WORK IF BOT IS ON ONE DISCORD SERVER
			channelID = msg.ChannelID
		}
		if !msg.Author.Bot && strings.HasPrefix(msg.Content, "!bb ") {
			cmd := msg.Content[4:]
			var op *mcserver.ServerRequestOp
			for _, matcherHandler := range MatcherHandlers {
				op = matcherHandler(cmd)
				if op != nil {
					fmt.Println("<- request: " + cmd)
					serverRequests <- op
					return
				}
			}
			discordResponses <- "ERROR: command \"" + cmd + "\" not recognized. get it together."
		}

	}

	client.On(disgord.EvtMessageCreate, handleMessage)

	fmt.Println("BOT IS LISTENING")

	go func() {
		for {
			discordMsg := <-discordResponses
			client.CreateMessage(bg, channelID, &disgord.CreateMessageParams{
				Content: discordMsg,
			})
		}
	}()
}
