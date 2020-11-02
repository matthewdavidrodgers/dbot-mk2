package defs

// ServerRequestOpCode is an int describing the operation type in a server request
type ServerRequestOpCode int

const (
	// Start describes a request to start an idle or crashed server
	Start ServerRequestOpCode = iota
	// Stop describes a request to stop a running server
	Stop
	// Kill describes a request to kill a running or starting server
	Kill
	// Logs describes a request to get the current logs of a server
	Logs
	// Status describes a request to get the current status of a server
	Status
	// Address describes a request to get the public DNS with which to connect to the server
	Address
	// Help describes a request to get the available commands and other help for the bot
	Help
	// Create describes a request to create a new world with params
	Create
	// List describes a request to list all existing worlds
	List
)

// ServerRequestOp is a unit describing an operation in a server request
type ServerRequestOp struct {
	Args map[string]string
	Code ServerRequestOpCode
}

// ServerResponseOpCode is an int describing the update type in a server response
type ServerResponseOpCode int

const (
	// Started describes a response to the server successfully starting
	Started ServerResponseOpCode = iota
	// Stopped describes a response to the server stopping (expected or not)
	Stopped
	// CreateWorldSuccess describes a response to the successful creation of a world
	CreateWorldSuccess
	// CreateWorldFailure describes a response to the unsuccessful creation of a world
	CreateWorldFailure
)

// ServerResponseOp is a unit describing an update in a server response
type ServerResponseOp struct {
	Args map[string]string
	Code ServerResponseOpCode
}
