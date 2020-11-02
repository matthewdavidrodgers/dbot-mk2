package mcserver

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/matthewdavidrodgers/dbot-mk2/defs"
	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

type serverAction func(m *manager, args map[string]string) string

var startServerRequestAction = func(m *manager, args map[string]string) string {
	if m.state == running || m.state == starting {
		return "ERROR: server is already running; you cannot start it"
	} else if m.state == stopping {
		return "ERROR: server is shutting down; wait for it to stop before restarting it"
	}

	requestedWorld, ok := args["_unnamed"]
	if !ok {
		return "ERROR: world name is missing. please supply as an unnamed option after the command. i.e. \"!bb start _my-world_\""
	}
	worldIsValid := false
	worlds, _ := getWorlds()
	for _, world := range worlds {
		if world.name == requestedWorld {
			worldIsValid = true
			break
		}
	}
	if !worldIsValid {
		return "ERROR: requested world is not valid. please supply an existing world or create a new one"
	}

	m.state = starting
	m.server = startServer(m.serverResponses, requestedWorld)
	return "SERVER IS STARTING. WAIT FOR START MESSAGE TO JOIN."
}

var stopServerRequestAction = func(m *manager, args map[string]string) string {
	if m.state != running || m.server == nil {
		return "ERROR: server is not running; it cannot be stopped"
	}
	m.state = stopping
	m.server.stop()
	return "STOPPING SERVER"
}

var killServerRequestAction = func(m *manager, args map[string]string) string {
	if m.server == nil {
		return "ERROR: no server to kill"
	}
	m.state = stopping
	m.server.kill()
	m.server = nil
	return "KILLING SERVER"
}

var statusServerRequestAction = func(m *manager, args map[string]string) string {
	var msg string
	if m.state == starting {
		msg = "SERVER IS STARTING UP. BE PATIENT. I WILL NOTIFY WHEN ITS READY."
	} else if m.state == idle {
		msg = "SERVER IS NOT RUNNING. TELL ME TO START IT. COME ON. I WANT YOU TO DO IT."
	} else if m.state == crashed {
		msg = "SHIT. SERVER HAS CRASHED. I HAVE NO ANSWERS. ONLY PAIN."
	} else if m.state == stopping {
		msg = "SERVER IS SHUTTING DOWN. IT IS A FAR BETTER REST THAT I GO TO THAN I HAVE EVER KNOWN."
	} else {
		// m.state == running
		msg = "SERVER IS RUNNING ON WORLD _" + m.server.worldName + "_. BLOC AWAY, MY BOIS.\n" + "server started on " + m.server.startedOn.String()
	}
	return msg
}

var logsServerRequestAction = func(m *manager, args map[string]string) string {
	if m.state == running && m.server != nil {
		return "ERROR: cannot get logs - server is running; stop and try again to see logs"
	}
	l, err := ioutil.ReadFile("bb-logs")
	if err != nil {
		fmt.Println("AH ERROR", err)
	}
	logLimit := 5
	logOffset := 0

	logLimitArg := args["l"]
	if parsedLogLimit, _ := strconv.Atoi(logLimitArg); parsedLogLimit > 0 {
		logLimit = parsedLogLimit
	}

	logOffsetArg := args["o"]
	if parsedLogOffset, _ := strconv.Atoi(logOffsetArg); parsedLogOffset > 0 {
		logOffset = parsedLogOffset
	}

	return utils.ReadLastLines(l, logLimit, logOffset)
}

var addressServerRequestAction = func(m *manager, args map[string]string) string {
	return "SERVER LISTENING FROM " + os.Getenv("PUBLIC_DNS")
}

var helpServerRequestAction = func(m *manager, args map[string]string) string {
	helpText := `Issue a command by messaging the bot with "!bb <your command> <options>"
e.g. if you wanted to start the server with the hyperion world: "!bb start hyperion"

COMMANDS`

	for _, c := range defs.Commands {
		helpText += "\n- " + c.HelpText
	}
	return helpText
}

var createServerRequestAction = func(m *manager, args map[string]string) string {
	if m.state != idle && m.state != crashed {
		return "ERROR: cannot create server while running. stop server and try again"
	}

	name, ok := args["name"]
	if !ok || name == "" {
		return "ERROR: world name is missing. please supply with the \"name\" option. e.g. -name=_my-new-world_"
	}
	valid := true
	worlds, _ := getWorlds()
	for _, world := range worlds {
		if world.name == name {
			valid = false
			break
		}
	}
	if !valid {
		return "ERROR: world \"" + name + "\" already exists. pick a new name"
	}

	mode, ok := args["mode"]
	if !ok {
		return "ERROR: mode is missing. please supply with the \"mode\" option. e.g. -mode=creative"
	}
	if mode != "creative" && mode != "survival" {
		return "ERROR: mode is not valid. options are \"creative\" and \"survival\""
	}

	go createWorld(m.serverResponses, name, mode)

	return "CREATING WORLD... WAIT FOR CONFIRMATION RESPONSE BEFORE STARTING"
}

var listServerRequestAction = func(m *manager, args map[string]string) string {
	worlds, err := getWorlds()
	if err != nil {
		fmt.Println(err)
		return "Uh oh. I... uh... could not list the worlds. Doesn't really sound good. But what do I know"
	}

	resp := "AVAILABLE WORLDS:\n"
	for _, world := range worlds {
		resp += fmt.Sprintf("\n%s (%s)", world.mode, world.mode)
	}

	resp += "\n\nStart a world with the \"start\" command i.e. \"!bb start _my-world_\""
	return resp
}

var serverRequestActions = map[defs.ServerRequestOpCode]serverAction{
	defs.Start:   startServerRequestAction,
	defs.Stop:    stopServerRequestAction,
	defs.Kill:    killServerRequestAction,
	defs.Status:  statusServerRequestAction,
	defs.Logs:    logsServerRequestAction,
	defs.Address: addressServerRequestAction,
	defs.Help:    helpServerRequestAction,
	defs.Create:  createServerRequestAction,
	defs.List:    listServerRequestAction,
}

var startedServerResponseAction = func(m *manager, args map[string]string) string {
	m.state = running
	return "SERVER IS READY. BLOC AWAY MY BOIS"
}

var stoppedServerResponseAction = func(m *manager, args map[string]string) string {
	m.server = nil
	if m.state != stopping {
		m.state = crashed
		return "SHIT. SERVER HAS CRASHED"
	}
	m.state = idle
	return "SERVER HAS STOPPED."
}

var createdWorldSuccessServerResonseAction = func(m *manager, args map[string]string) string {
	worldName := args["name"]
	return "WORLD \"" + worldName + "\" CREATED. START IF YOU DARE."
}

var createdWorldFailureServerResonseAction = func(m *manager, args map[string]string) string {
	return "ERROR: COULD NOT CREATE WORLD"
}

var serverResponseActions = map[defs.ServerResponseOpCode]serverAction{
	defs.Started:            startedServerResponseAction,
	defs.Stopped:            stoppedServerResponseAction,
	defs.CreateWorldFailure: createdWorldFailureServerResonseAction,
	defs.CreateWorldSuccess: createdWorldSuccessServerResonseAction,
}
