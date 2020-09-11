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
	starting = "starting"
	running  = "running"
	stopping = "stopping"
	crashed  = "crashed"
)

type serverRequestOpCode int

const (
	start serverRequestOpCode = iota
	stop
	kill
	logs
	status
)

type serverResponseOpCode int

const (
	started serverResponseOpCode = iota
	stopped
)

type serverRequestOp struct {
	op   serverRequestOpCode
	args map[string]string
}

type serverResponseOp struct {
	op serverResponseOpCode
}

type server struct {
	StartedOn time.Time
	Stop      func()
	Kill      func()
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

func startServer(notify chan<- *serverResponseOp) *server {
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
	portPollSucceeded := make(chan bool)
	abortPortPolling := make(chan bool)

	go func() {
		logFile.WriteString("\n\n=== BEGIN BB SESSION " + now.String() + " ===\n\n\n")
		serverCmd.Start()
		serverCmd.Wait()

		logFile.Close()
		notify <- &serverResponseOp{op: stopped}
	}()

	go func() {
		for {
			pingPortCmd := exec.Command("/bin/sh", "-c", "sudo lsof -i -P -n | grep 'TCP \\*:25565 (LISTEN)'")
			resp, err := pingPortCmd.CombinedOutput()
			fmt.Println(resp, err)

			bound := err == nil && len(resp) > 0
			portPollSucceeded <- bound
			if bound {
				fmt.Println("polled for port - success!")
				return
			}
			fmt.Println("polled for port - failed")
			time.Sleep(5 * time.Second)

		}

	}()

	go func() {
		for {
			select {
			case portWasBound := <-portPollSucceeded:
				if portWasBound {
					notify <- &serverResponseOp{op: started}
					return
				}
			case <-abortPortPolling:
				return
			}
		}
	}()

	return &server{
		StartedOn: now,
		Stop: func() {
			serverInputPipe.Write([]byte("stop\n"))
			serverInputPipe.Close()
		},
		Kill: func() {
			abortPortPolling <- true
			serverCmd.Process.Kill()
		},
	}

}

func makeServerManager(serverRequests <-chan *serverRequestOp, discordResponses chan<- string) {
	var state serverState = idle
	var server *server
	serverResponses := make(chan *serverResponseOp)

	incomingArrow := "-> "
	outgoingArrow := "<- "
	for {
		select {
		case serverRequest := <-serverRequests:
			switch serverRequest.op {
			case start:
				fmt.Println(incomingArrow + "request: start")
				if state == idle || state == crashed {
					state = starting
					server = startServer(serverResponses)
					msg := "SERVER IS STARTING. WAIT FOR START MESSAGE TO JOIN."

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				} else if state == running || state == starting {
					msg := "ERROR: server is already running; you cannot start it"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				} else if state == stopping {
					msg := "ERROR: server is shutting down; wait for it to stop before restarting it"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				}
				break
			case stop:
				fmt.Println(incomingArrow + "request: stop")
				if state == running && server != nil {
					state = stopping
					server.Stop()
					msg := "STOPPING SERVER"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				} else {
					msg := "ERROR: server is not running; it cannot be stopped"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				}
				break
			case kill:
				if server != nil {
					state = stopping
					server.Kill()
					server = nil
				}
				fmt.Println(incomingArrow, "request: kill")
				break
			case status:
				var msg string
				fmt.Println(incomingArrow + "request: status")
				if state == running {
					msg = "SERVER IS RUNNING. BLOC AWAY, MY BOIS.\n" + "server started on " + server.StartedOn.String()
				} else if state == starting {
					msg = "SERVER IS STARTING UP. BE PATIENT. I WILL NOTIFY WHEN ITS READY."
				} else if state == idle {
					msg = "SERVER IS NOT RUNNING. TELL ME TO START IT. COME ON. I WANT YOU TO DO IT."
				} else if state == crashed {
					msg = "SHIT. SERVER HAS CRASHED. I HAVE NO ANSWERS. ONLY PAIN."
				} else if state == stopping {
					msg = "SERVER IS SHUTTING DOWN. IT IS A FAR BETTER REST THAT I GO TO THAN I HAVE EVER KNOWN."
				}
				fmt.Println(outgoingArrow + "STATUS: " + msg)
				discordResponses <- msg
				break
			case logs:
				fmt.Println(incomingArrow + "request: logs")
				if state == running && server != nil {
					// TODO: figure this shit out
					msg := "ERROR: cannot get logs - server is running; stop and try again to see logs"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				} else {
					l, err := ioutil.ReadFile("bb-logs")
					if err != nil {
						fmt.Println("AH ERROR", err)
					}
					logLimit := 5
					logOffset := 0

					logLimitArg := serverRequest.args["l"]
					if parsedLogLimit, _ := strconv.Atoi(logLimitArg); parsedLogLimit > 0 {
						logLimit = parsedLogLimit
					}

					logOffsetArg := serverRequest.args["o"]
					if parsedLogOffset, _ := strconv.Atoi(logOffsetArg); parsedLogOffset > 0 {
						logOffset = parsedLogOffset
					}

					truncatedLogs := readLastLines(l, logLimit, logOffset)
					msg := "sending logs:\n" + truncatedLogs + "\n"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- truncatedLogs
				}
				break
			}
		case serverResponse := <-serverResponses:
			switch serverResponse.op {
			case started:
				state = running
				fmt.Println("<> server has started")

				msg := "SERVER IS READY. BLOC AWAY MY BOIS."
				fmt.Println(outgoingArrow + msg)
				discordResponses <- msg
			case stopped:
				fmt.Println("<> server has stopped")
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
				discordResponses <- msg
				break
			}

		}
	}
}

func makeBotManager(serverRequests chan<- *serverRequestOp, discordResponses chan string) {
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
			var op *serverRequestOp
			if cmd == "start" {
				op = &serverRequestOp{op: start}
			} else if cmd == "stop" {
				op = &serverRequestOp{op: stop}
			} else if cmd == "kill" {
				op = &serverRequestOp{op: kill}
			} else if cmd == "status" {
				op = &serverRequestOp{op: status}
			} else if strings.HasPrefix(cmd, "logs") {
				op = &serverRequestOp{op: logs}
				argString := cmd[4:]
				var validFlags = []string{"l", "o"}
				compiledArgs, err := parseArgString(argString, validFlags)
				if err != nil {
					op = nil
					fmt.Println(err)
				} else {
					op.args = compiledArgs
				}
			}

			if op != nil {
				serverRequests <- op
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

func main() {
	serverRequests := make(chan *serverRequestOp)
	discordResponses := make(chan string)

	go makeServerManager(serverRequests, discordResponses)
	makeBotManager(serverRequests, discordResponses)
}
