package mcserver

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

type serverState string

const (
	idle     = "idle"
	starting = "starting"
	running  = "running"
	stopping = "stopping"
	crashed  = "crashed"
)

type server struct {
	startedOn time.Time
	stop      func()
	kill      func()
}

func startServer(notify chan<- *ServerResponseOp) *server {
	serverCmd := exec.Command("java", "-Xmx1024M", "-Xms512M", "-jar", "server.jar", "--nogui", "--universe", "bb-worlds", "--world", "hyperion")
	pwd, err := os.Getwd()
	utils.Check(err)
	serverCmd.Dir = pwd

	logFile, err := os.OpenFile("bb-logs", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	utils.Check(err)

	serverCmd.Stdout = logFile
	serverCmd.Stderr = logFile
	serverInputPipe, err := serverCmd.StdinPipe()
	utils.Check(err)

	now := time.Now()
	portPollSucceeded := make(chan bool)
	abortPortPolling := make(chan bool)

	go func() {
		logFile.WriteString("\n\n=== BEGIN BB SESSION " + now.String() + " ===\n\n\n")
		serverCmd.Start()
		serverCmd.Wait()

		logFile.Close()
		notify <- &ServerResponseOp{Code: stopped}
	}()

	go func() {
		for {
			pingPortCmd := exec.Command("/bin/sh", "-c", "sudo lsof -i -P -n | grep 'TCP \\*:25565 (LISTEN)'")
			resp, err := pingPortCmd.CombinedOutput()

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
					notify <- &ServerResponseOp{Code: started}
					return
				}
			case <-abortPortPolling:
				return
			}
		}
	}()

	return &server{
		startedOn: now,
		stop: func() {
			serverInputPipe.Write([]byte("stop\n"))
			serverInputPipe.Close()
		},
		kill: func() {
			abortPortPolling <- true
			serverCmd.Process.Kill()
		},
	}

}

// MakeServerManager listens to the serverRequest channel and performs ops against a mc server, sending string updates to the discordMessages channel
func MakeServerManager(serverRequests <-chan *ServerRequestOp, discordResponses chan<- string) {
	var state serverState = idle
	var server *server
	serverResponses := make(chan *ServerResponseOp)

	outgoingArrow := "<- "
	for {
		select {
		case serverRequest := <-serverRequests:
			switch serverRequest.Code {
			case Start:
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
			case Stop:
				if state == running && server != nil {
					state = stopping
					server.stop()
					msg := "STOPPING SERVER"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				} else {
					msg := "ERROR: server is not running; it cannot be stopped"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- msg
				}
				break
			case Kill:
				if server != nil {
					state = stopping
					server.kill()
					server = nil
				}
				break
			case Status:
				var msg string
				if state == running {
					msg = "SERVER IS RUNNING. BLOC AWAY, MY BOIS.\n" + "server started on " + server.startedOn.String()
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
			case Logs:
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

					logLimitArg := serverRequest.Args["l"]
					if parsedLogLimit, _ := strconv.Atoi(logLimitArg); parsedLogLimit > 0 {
						logLimit = parsedLogLimit
					}

					logOffsetArg := serverRequest.Args["o"]
					if parsedLogOffset, _ := strconv.Atoi(logOffsetArg); parsedLogOffset > 0 {
						logOffset = parsedLogOffset
					}

					truncatedLogs := utils.ReadLastLines(l, logLimit, logOffset)
					msg := "sending logs:\n" + truncatedLogs + "\n"

					fmt.Println(outgoingArrow + msg)
					discordResponses <- truncatedLogs
				}
				break
			}
		case serverResponse := <-serverResponses:
			switch serverResponse.Code {
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
