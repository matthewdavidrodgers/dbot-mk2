package mcserver

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/matthewdavidrodgers/dbot-mk2/defs"
	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

type server struct {
	startedOn time.Time
	worldName string
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
	serverResponses chan *defs.ServerResponseOp
}

type bbWorld struct {
	name string
	mode string
}

func getWorlds() ([]bbWorld, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.ReadDir(filepath.Join(pwd, "bb-worlds"))
	if err != nil {
		return nil, err
	}

	worlds := make([]bbWorld, 0)
	for _, world := range dir {
		if world.IsDir() {
			name := world.Name()
			path := filepath.Join(pwd, "bb-worlds", name, "server.properties")
			mode, err := utils.GetNamedValueInTextFile(path, "gamemode")
			if err != nil {
				return nil, err
			}

			worlds = append(worlds, bbWorld{name: name, mode: mode})
		}
	}

	return worlds, nil
}

func createWorld(notify chan<- *defs.ServerResponseOp, name string, mode string) {
	pwd, err := os.Getwd()
	if err != nil {
		notify <- &defs.ServerResponseOp{Code: defs.CreateWorldFailure}
		return
	}
	path := filepath.Join(pwd, "bb-worlds", name)
	err = os.Mkdir(path, 0755)
	if err != nil {
		notify <- &defs.ServerResponseOp{Code: defs.CreateWorldFailure}
		return
	}

	jarFileLocation := filepath.Join(pwd, "server.jar")

	createCmd := exec.Command("java", "-Xmx1024M", "-Xms512M", "-jar", jarFileLocation, "--nogui", "--initSettings")
	createCmd.Dir = path

	output, err := createCmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		notify <- &defs.ServerResponseOp{Code: defs.CreateWorldFailure}
		return
	}

	utils.ReplaceNamedValueInTextFile(filepath.Join(path, "server.properties"), "gamemode", mode)
	utils.ReplaceNamedValueInTextFile(filepath.Join(path, "eula.txt"), "eula", "true")

	op := defs.ServerResponseOp{Code: defs.CreateWorldSuccess}
	args := map[string]string{"name": name}
	op.Args = args
	notify <- &op
}

func startServer(notify chan<- *defs.ServerResponseOp, world string) *server {
	serverCmd := exec.Command("java", "-Xmx1024M", "-Xms512M", "-jar", "../../server.jar", "--nogui")
	pwd, err := os.Getwd()
	utils.Check(err)
	serverCmd.Dir = filepath.Join(pwd, "bb-worlds", world)

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
		notify <- &defs.ServerResponseOp{Code: defs.Stopped}
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
					notify <- &defs.ServerResponseOp{Code: defs.Started}
					return
				}
			case <-abortPortPolling:
				return
			}
		}
	}()

	return &server{
		startedOn: now,
		worldName: world,
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
func MakeServerManager(serverRequests <-chan *defs.ServerRequestOp, discordResponses chan<- string) {
	serverResponses := make(chan *defs.ServerResponseOp)
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
