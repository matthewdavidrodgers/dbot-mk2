package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
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

type serverOpCode int

const (
	reqStart serverOpCode = iota
	reqStop
	reqStatus
	reqLogs
	stopped
	noOp
)

type serverOp struct {
	op   serverOpCode
	args map[string]string
}

type server struct {
	StartedOn time.Time
	Stop      func()
}

func parseArgString(argString string, argFlags []string) (map[string]string, error) {
	args := make(map[string]string, len(argFlags))
	e := errors.New("ERROR: malformed argString")

	idx := 0
	for idx < len(argString) {
		charByte := argString[idx]
		char := string(charByte)
		if char == " " {
			idx++
			continue
		}
		if char == "-" {
			if idx+1 == len(argString) {
				return nil, e
			}
			flagByte := argString[idx+1]
			if !((flagByte >= 65 && flagByte <= 90) || (flagByte >= 97 && flagByte <= 122)) {
				return nil, e
			}
			flag := string(flagByte)

			if idx+2 == len(argString) || string(argString[idx+2]) != "=" {
				return nil, e
			}

			innerIdx := idx + 3
			flagValue := make([]byte, 0)
			for innerIdx < len(argString) {
				innerCharByte := argString[innerIdx]
				innerChar := string(innerCharByte)

				if innerChar == " " {
					break
				}
				flagValue = append(flagValue, innerCharByte)
				innerIdx++
			}
			if len(flagValue) == 0 {
				return nil, e
			}
			args[flag] = string(flagValue)
			idx = innerIdx + 1
			continue
		}
		return nil, e
	}
	return args, nil
}

func readLastLines(bytes []byte, l int, o int) string {
	lines := strings.Split(string(bytes), "\n")
	firstLineIndex := len(lines) - l - o
	lastLineIndex := len(lines) - o
	if firstLineIndex < 0 {
		firstLineIndex = 0
	}
	return strings.Join(lines[firstLineIndex:lastLineIndex], "\n")
}

func startServer(signals chan<- *serverOp) *server {
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
		signals <- &serverOp{op: stopped}
	}()

	return &server{
		StartedOn: now,
		Stop: func() {
			serverInputPipe.Write([]byte("stop\n"))
			serverInputPipe.Close()
		},
	}

}

func makeServerManager(serverChan chan *serverOp, discordChan chan<- string) {
	var state serverState = idle
	var server *server

	incomingArrow := "-> "
	outgoingArrow := "<- "
	for {
		serverOperation := <-serverChan
		switch serverOperation.op {
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
				logLimit := 5
				logOffset := 0

				logLimitArg := serverOperation.args["l"]
				if parsedLogLimit, _ := strconv.Atoi(logLimitArg); parsedLogLimit > 0 {
					logLimit = parsedLogLimit
				}

				logOffsetArg := serverOperation.args["o"]
				if parsedLogOffset, _ := strconv.Atoi(logOffsetArg); parsedLogOffset > 0 {
					logOffset = parsedLogOffset
				}

				truncatedLogs := readLastLines(logs, logLimit, logOffset)
				msg := "sending logs:\n" + truncatedLogs + "\n"

				fmt.Println(outgoingArrow + msg)
				discordChan <- truncatedLogs
			}
			break
		case stopped:
			fmt.Println(incomingArrow + "op: stopped")
			var msg string
			if state == stopping {
				state = idle
				msg = "SERVER HAS STOPPED."
			} else {
				state = crashed
				msg = "SHIT. SERVER HAS CRASHED."
			}
			server = nil

			fmt.Println(outgoingArrow + msg)
			discordChan <- msg
			break
		}
	}
}

func makeBotManager(serverChan chan<- *serverOp, discordChan chan string) {
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
			op := noOp
			var args map[string]string = nil
			if cmd == "start" {
				op = reqStart
			} else if cmd == "stop" {
				op = reqStop
			} else if cmd == "status" {
				op = reqStatus
			} else if strings.HasPrefix(cmd, "logs") {
				op = reqLogs
				argString := cmd[4:]
				var validFlags = []string{"l", "o"}
				compiledArgs, err := parseArgString(argString, validFlags)
				if err != nil {
					fmt.Println(err)
					op = noOp
				}
				args = compiledArgs
			}

			if op != noOp {
				serverChan <- &serverOp{op: op, args: args}
			} else {
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
	serverChan := make(chan *serverOp)
	discordChan := make(chan string)

	go makeServerManager(serverChan, discordChan)
	makeBotManager(serverChan, discordChan)
}
