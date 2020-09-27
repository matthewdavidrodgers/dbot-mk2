package mcserver

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

type server struct {
	startedOn time.Time
	stop      func()
	kill      func()
}

type serverStateCode int

const (
	idle serverStateCode = iota
	starting
	running
	stopping
	crashed
)

type manager struct {
	state           serverStateCode
	server          *server
	serverResponses chan *ServerResponseOp
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
	serverResponses := make(chan *ServerResponseOp)
	serverManager := &manager{state: idle, server: nil, serverResponses: serverResponses}

	go func() {
		outgoingArrow := "<- "
		for {
			var action serverAction
			var ok bool
			var args map[string]string

			select {
			case serverRequest := <-serverRequests:
				args = serverRequest.Args
				action, ok = serverRequestActions[serverRequest.Code]
				break
			case serverResponse := <-serverResponses:
				args = serverResponse.Args
				action, ok = serverResponseActions[serverResponse.Code]
			}

			if !ok {
				fmt.Println("Hm... unknown action requested")
				continue
			}
			responseMsg := action(serverManager, args)
			fmt.Println(outgoingArrow + responseMsg)
			discordResponses <- responseMsg
		}
	}()
}
