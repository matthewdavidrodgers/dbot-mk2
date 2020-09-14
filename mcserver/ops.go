package mcserver

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
	// NoOp is an operation that is ignored by the server manager
	NoOp
)

// ServerRequestOp is a unit describing an operation in a server request
type ServerRequestOp struct {
	Code ServerRequestOpCode
	Args map[string]string
}

// ServerResponseOpCode is an int describing the update type in a server response
type ServerResponseOpCode int

const (
	started ServerResponseOpCode = iota
	stopped
)

// ServerResponseOp is a unit describing an update in a server response
type ServerResponseOp struct {
	Code ServerResponseOpCode
}
