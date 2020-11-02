package defs

// MessageCommand is configuration data required to parse a discord message into a server operation
type MessageCommand struct {
	Command         string
	FlagArgs        []string
	AllowUnnamedArg bool
	RequestCode     ServerRequestOpCode
	HelpText        string
}

// Commands is a list of all available Commands
var Commands = []MessageCommand{
	{
		Command:         "start",
		AllowUnnamedArg: true,
		RequestCode:     Start,
		HelpText:        "start _world-name_ : start the server on the specified world. it won't immediately be available - the bot will message you when it's ready",
	},
	{
		Command:     "stop",
		RequestCode: Stop,
		HelpText:    "stop : safely stop a running server",
	},
	{
		Command:     "kill",
		RequestCode: Kill,
		HelpText:    "kill : unsafely stop a running or starting server (be careful, this could corrupt the minecraft world)",
	},
	{
		Command:     "status",
		RequestCode: Status,
		HelpText:    "status : report on the status of the server",
	},
	{
		Command:     "address",
		RequestCode: Address,
		HelpText:    "address : get the public dns address of the server (what you'll use to connect to it)",
	},
	{
		Command:     "help",
		RequestCode: Help,
		HelpText:    "help : list available Commands",
	},
	{
		Command:     "logs",
		RequestCode: Logs,
		FlagArgs:    []string{"l", "o"},
		HelpText:    "logs : print out a list of the most recent logs. control with flags _l_ (limit) and _o_ (offset). i.e. \"!bb logs -l=10 -o=15\"",
	},
	{
		Command:     "create",
		RequestCode: Create,
		FlagArgs:    []string{"name", "mode"},
		HelpText:    "create : create a new world. required params: _name_ and _mode_. i.e. \"!bb create -name=my-new-world -mode=creative\"",
	},
	{
		Command:     "list",
		RequestCode: List,
		HelpText:    "list : list the existing worlds",
	},
}
