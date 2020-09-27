package dbot

import (
	"fmt"

	"github.com/matthewdavidrodgers/dbot-mk2/mcserver"
	"github.com/matthewdavidrodgers/dbot-mk2/utils"
)

type matcherHandler func(cmd string) *mcserver.ServerRequestOp

var start matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	if cmd == "start" {
		return &mcserver.ServerRequestOp{Code: mcserver.Start}
	}
	return nil
}

var stop matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	if cmd == "stop" {
		return &mcserver.ServerRequestOp{Code: mcserver.Stop}
	}
	return nil
}

var kill matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	if cmd == "kill" {
		return &mcserver.ServerRequestOp{Code: mcserver.Kill}
	}
	return nil
}

var status matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	if cmd == "status" {
		return &mcserver.ServerRequestOp{Code: mcserver.Status}
	}
	return nil
}

var address matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	if cmd == "address" {
		return &mcserver.ServerRequestOp{Code: mcserver.Address}
	}
	return nil
}

var help matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	if cmd == "help" {
		return &mcserver.ServerRequestOp{Code: mcserver.Help}
	}
	return nil
}

var logs matcherHandler = func(cmd string) *mcserver.ServerRequestOp {
	op := &mcserver.ServerRequestOp{Code: mcserver.Logs}
	argString := cmd[4:]
	var validFlags = []string{"l", "o"}
	compiledArgs, err := utils.ParseArgString(argString, validFlags)
	if err != nil {
		op = nil
		fmt.Println(err)
	} else {
		op.Args = compiledArgs
	}
	return op
}

// MatcherHandlers represents a list of actions that the server can recognize
var MatcherHandlers = []matcherHandler{
	start,
	stop,
	kill,
	status,
	address,
	help,
	logs,
}
