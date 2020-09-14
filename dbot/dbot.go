package dbot

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/matthewdavidrodgers/dbot-mk2/mcserver"
	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

var helpText = `Issue a command by messaging the bot with "!bb <your command>"
e.g. if you wanted to start the server: "!bb start"

COMMANDS
- start : start the server if it hasn't been running. it won't immediately be available - the bot will message you when its ready
- stop : safely stop a running server
- kill : unsafely stop a running or starting server (be careful, this could corrupt the minecraft world)
- status : report on the status of the server
- address : get the public dns address of the server (what you'll use to connect to it)
- help : list available commands 
`

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
			if cmd == "start" {
				op = &mcserver.ServerRequestOp{Code: mcserver.Start}
			} else if cmd == "stop" {
				op = &mcserver.ServerRequestOp{Code: mcserver.Stop}
			} else if cmd == "kill" {
				op = &mcserver.ServerRequestOp{Code: mcserver.Kill}
			} else if cmd == "status" {
				op = &mcserver.ServerRequestOp{Code: mcserver.Status}
			} else if cmd == "address" {
				dns := os.Getenv("PUBLIC_DNS")
				discordResponses <- "SERVER LISTENING FROM " + dns
				op = &mcserver.ServerRequestOp{Code: mcserver.NoOp}
			} else if cmd == "help" {
				discordResponses <- helpText
				op = &mcserver.ServerRequestOp{Code: mcserver.NoOp}
			} else if strings.HasPrefix(cmd, "logs") {
				op = &mcserver.ServerRequestOp{Code: mcserver.Logs}
				argString := cmd[4:]
				var validFlags = []string{"l", "o"}
				compiledArgs, err := utils.ParseArgString(argString, validFlags)
				if err != nil {
					op = nil
					fmt.Println(err)
				} else {
					op.Args = compiledArgs
				}
			}

			if op != nil {
				fmt.Println("<- request: " + cmd)
				if op.Code != mcserver.NoOp {
					serverRequests <- op
				}
			} else {
				discordResponses <- "ERROR: command \"" + cmd + "\" not recognized. get it together."
			}
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
