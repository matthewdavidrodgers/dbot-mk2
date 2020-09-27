package mcserver

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

const helpText = `Issue a command by messaging the bot with "!bb <your command>"
e.g. if you wanted to start the server: "!bb start"

COMMANDS
- start : start the server if it hasn't been running. it won't immediately be available - the bot will message you when its ready
- stop : safely stop a running server
- kill : unsafely stop a running or starting server (be careful, this could corrupt the minecraft world)
- status : report on the status of the server
- address : get the public dns address of the server (what you'll use to connect to it)
- help : list available commands 
`

type serverAction func(m *manager, args map[string]string) string

var startServerRequestAction = func(m *manager, args map[string]string) string {
	if m.state == running || m.state == starting {
		return "ERROR: server is already running; you cannot start it"
	} else if m.state == stopping {
		return "ERROR: server is shutting down; wait for it to stop before restarting it"
	}

	m.state = starting
	m.server = startServer(m.serverResponses)
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
		msg = "SERVER IS RUNNING. BLOC AWAY, MY BOIS.\n" + "server started on " + m.server.startedOn.String()
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
	return helpText
}

var serverRequestActions = map[ServerRequestOpCode]serverAction{
	Start:   startServerRequestAction,
	Stop:    stopServerRequestAction,
	Kill:    killServerRequestAction,
	Status:  statusServerRequestAction,
	Logs:    logsServerRequestAction,
	Address: addressServerRequestAction,
	Help:    helpServerRequestAction,
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

var serverResponseActions = map[ServerResponseOpCode]serverAction{
	started: startedServerResponseAction,
	stopped: stoppedServerResponseAction,
}
