package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type serverState string

const (
	idle     = "idle"
	running  = "running"
	stopping = "stopping"
	crashed  = "crashed"
)

type serverOp int

const (
	reqStart serverOp = iota
	reqStop
	reqStatus
	reqLogs
	stopped
)

type server struct {
	StartedOn time.Time
	Stop      func()
}

func readLastNLines(bytes []byte, n int) string {
	lines := strings.Split(string(bytes), "\n")
	var nLines []string
	if len(lines) <= n {
		nLines = lines
	} else {
		nLines = lines[len(lines)-n-1:]
	}
	return strings.Join(nLines, "\n")
}

func startServer(signals chan<- serverOp) *server {
	serverCmd := exec.Command("java", "-Xmx1024M", "-Xms1024M", "-jar", "server.jar", "nogui")
	pwd, err := os.Getwd()
	check(err)
	serverCmd.Dir = pwd

	logFile, err := os.OpenFile("bb-logs", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	check(err)

	serverCmd.Stdout = logFile
	serverCmd.Stderr = logFile
	serverInputPipe, err := serverCmd.StdinPipe()
	check(err)

	now := time.Now()

	go func() {
		logFile.WriteString("\n\n=== BEGIN BB SESSION " + now.String() + " ===\n\n\n")
		serverCmd.Start()
		serverCmd.Wait()

		logFile.Close()
		signals <- stopped
	}()

	return &server{
		StartedOn: now,
		Stop: func() {
			serverInputPipe.Write([]byte("stop\n"))
			serverInputPipe.Close()
		},
	}

}

func makeServerManager(serverChan chan serverOp, discordChan chan<- string) {
	var state serverState = idle
	var server *server

	incomingArrow := "-> "
	outgoingArrow := "<- "
	for {
		serverOp := <-serverChan
		switch serverOp {
		case reqStart:
			fmt.Println(incomingArrow + "op: reqStart")
			if state == idle || state == crashed {
				state = running
				server = startServer(serverChan)
				msg := "SERVER STARTED"

				fmt.Println(outgoingArrow + msg)
				discordChan <- msg
			} else if state == running {
				msg := "ERROR: server is already running; you cannot start it"

				fmt.Println(outgoingArrow + msg)
				discordChan <- msg
			} else if state == stopping {
				msg := "ERROR: server is shutting down; wait for it to stop before restarting it"

				fmt.Println(outgoingArrow + msg)
				discordChan <- msg
			}
			break
		case reqStop:
			fmt.Println(incomingArrow + "op: reqStop")
			if state == running && server != nil {
				state = stopping
				server.Stop()
				msg := "SERVER STOPPED"

				fmt.Println(outgoingArrow + msg)
				discordChan <- msg
			} else {
				msg := "ERROR: server is not running; it cannot be stopped"

				fmt.Println(outgoingArrow + msg)
				discordChan <- msg
			}
			break
		case reqStatus:
			fmt.Println(incomingArrow + "op: reqStatus")
			if state == running {
				msg := "SERVER IS RUNNING. BLOC AWAY, MY BOIS.\n" + "server started on " + server.StartedOn.String()

				fmt.Println(outgoingArrow + "STATUS: " + msg)
				discordChan <- msg
			} else if state == idle {
				msg := "SERVER IS NOT RUNNING. TELL ME TO START IT. COME ON. I WANT YOU TO DO IT."

				fmt.Println(outgoingArrow + "STATUS: " + msg)
				discordChan <- msg
			} else if state == crashed {
				msg := "SHIT. SERVER HAS CRASHED. I HAVE NO ANSWERS. ONLY PAIN."

				fmt.Println(outgoingArrow + "STATUS: " + msg)
				discordChan <- msg
			} else if state == stopping {
				msg := "SERVER IS SHUTTING DOWN. IT IS A FAR BETTER REST THAT I GO TO THAN I HAVE EVER KNOWN."

				fmt.Println(outgoingArrow + "STATUS: " + msg)
				discordChan <- msg
			}
			break
		case reqLogs:
			fmt.Println(incomingArrow + "op: reqLogs")
			if state == running && server != nil {
				// TODO: figure this shit out
				msg := "ERROR: cannot get logs - server is running; stop and try again to see logs"

				fmt.Println(outgoingArrow + msg)
				discordChan <- msg
			} else {
				logs, err := ioutil.ReadFile("bb-logs")
				if err != nil {
					fmt.Println("AH ERROR", err)
				}
				truncatedLogs := readLastNLines(logs, 5)
				msg := "sending logs:\n" + truncatedLogs + "\n"

				fmt.Println(outgoingArrow + msg)
				discordChan <- truncatedLogs
			}
			break
		case stopped:
			fmt.Println(incomingArrow + "op: stopped")
			if state == stopping {
				fmt.Println(outgoingArrow + "server has stopped")
				state = idle
			} else {
				fmt.Println(outgoingArrow + "server has CRASHED")
				state = crashed
			}
			server = nil
			break
		}
	}
}

func makeBotManager(serverChan chan<- serverOp, discordChan chan string) {
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
			switch cmd {
			case "start":
				serverChan <- reqStart
				break
			case "stop":
				serverChan <- reqStop
				break
			case "status":
				serverChan <- reqStatus
				break
			case "logs":
				// TODO: compile flags for limit and offset (e.g. -l=10 -o=15)
				serverChan <- reqLogs
				break
			default:
				discordChan <- "ERROR: command \"" + cmd + "\" not recognized. get it together."
			}
		}
	}
	client.On(disgord.EvtMessageCreate, handleMessage)

	fmt.Println("BOT IS LISTENING")

	go func() {
		for {
			discordMsg := <-discordChan
			client.CreateMessage(bg, channelID, &disgord.CreateMessageParams{
				Content: discordMsg,
			})
		}
	}()
}

func main() {
	serverChan := make(chan serverOp)
	discordChan := make(chan string)

	go makeServerManager(serverChan, discordChan)
	makeBotManager(serverChan, discordChan)
}
