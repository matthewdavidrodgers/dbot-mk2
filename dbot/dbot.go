package dbot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/matthewdavidrodgers/dbot-mk2/defs"
	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

func parseOp(message string, mcs []defs.MessageCommand) (*defs.ServerRequestOp, error) {
	for _, mc := range mcs {
		if message == mc.Command || strings.HasPrefix(message, mc.Command+" ") {
			argString := message[len(mc.Command):]

			compiledArgs, err := utils.ParseArgString(argString, mc.FlagArgs, mc.AllowUnnamedArg)
			if err != nil {
				fmt.Println(err)
				e, ok := err.(*utils.InvalidFlagError)
				if ok {
					return nil, fmt.Errorf("flag \"%s\" is not allow for command \"%s\"", e.Found, mc.Command)
				}
				return nil, errors.New("i have literally no idea what that means")
			}

			return &defs.ServerRequestOp{Code: mc.RequestCode, Args: compiledArgs}, nil
		}
	}
	return nil, fmt.Errorf("command \"%s\" is not recognized. get it together", message)
}

// MakeBotManager starts discord bot that listens to incoming messages, and sends ServerRequestOps when a valid
// command is requested. it also sends messages back to the discord server based on the messages provided by the
// discordResponses channel
func MakeBotManager(serverRequests chan<- *defs.ServerRequestOp, discordResponses chan string) {
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
			op, err := parseOp(cmd, defs.Commands)
			if err != nil {
				discordResponses <- "ERROR: " + err.Error()
				return
			}
			serverRequests <- op
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
