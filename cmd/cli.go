package cmd

import (
	"fmt"
	"github.com/fzft/go-mock-redis/deps/hredis"
	"github.com/fzft/go-mock-redis/deps/linenoise"
	"github.com/mattn/go-isatty"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	RedisVersion    = "255.255.255"
	RedisVersionNum = 0x00ffffff

	RedisCliKeepAliveInternal  = 15
	RedisCliDefaultPipeTimeout = 30
	RedisCliHisFileEnv         = "REDISCLI_HISTFILE"
	RedisCliHisFileDefault     = ".rediscli_history"
	RedisCliRCFileEnv          = "REDISCLI_RCFILE"
	RedisCliRCFileDefault      = ".redisclirc"
	RedisCliAuthEnv            = "REDISCLI_AUTH"
)

type CliConnectFlag int

var context *hredis.RedisContext

const (
	CCForce CliConnectFlag = 1 << iota // Re-connect if already connected.
	CCQuiet                            // Don't show non-error messages.
)

type OutputMode uint8

const (
	OutputStandard = iota
	OutputRaw
	OutputJson
	OutputQuotedJson
)

type CliSSLConfig struct {
	// Use SSL/TLS for connection (default: false).
	caserts string

	// CA certificates directory.
	casertsDir string

	skipCasertsValidation bool

	// Use client certificate authentication (default: false).
	key string

	// Client cipher list.
	ciphers string
}

type CliConnInfo struct {
	hostIp     string
	hostPort   int
	inputDbNum int
	auth       string
	user       string
}

type RedisCliCfg struct {
	connInfo              *CliConnInfo
	hostSocket            string
	tls                   bool
	sslCfg                *CliSSLConfig
	repeat                int64
	dbNum                 int
	interactive           bool
	shutdown              bool
	inMulti               bool
	eval                  string
	evalLDB               bool
	evalLDBEnd            bool
	clusterMode           bool
	clusterReissueCommand bool
	pubsubMode            bool
	prompt                string
	serverVersion         string
	output                OutputMode
	lastReply             *hredis.RedisReply

	resp2        int //value of 1: specified explicitly,value of 2: specified implicitly
	resp3        int //value of 1: specified explicitly,value of 2: specified implicitly
	currentResp3 bool
}

type RedisCli struct {
	config *RedisCliCfg
}

func (cli *RedisCli) Version(gitSHA1, gitDirty string) string {
	var version string
	// Add git commit and working tree status when available
	if sha1Int, err := strconv.ParseInt(gitSHA1, 16, 64); err == nil && sha1Int != 0 {
		version = fmt.Sprintf("%s (git:%s", version, gitSHA1)
		if dirtyInt, err := strconv.ParseInt(gitDirty, 10, 64); err == nil && dirtyInt != 0 {
			version = fmt.Sprintf("%s-dirty", version)
		}
		version = fmt.Sprintf("%s)", version)
	}

	return version
}

func (cli *RedisCli) Usage(gitSHA1, gitDirty string, err bool) {
	var out io.Writer
	version := cli.Version(gitSHA1, gitDirty)
	if err {
		out = os.Stderr
	} else {
		out = os.Stdout
	}

	fmt.Fprintf(out, `"redis-cli %s\n"
"\n"
"Usage: redis-cli [OPTIONS] [cmd [arg [arg ...]]]\n"
"  -h <hostname>      Server hostname (default: 127.0.0.1).\n"
"  -p <port>          Server port (default: 6379).\n"
"  -s <socket>        Server socket (overrides hostname and port).\n"
"  -a <password>      Password to use when connecting to the server.\n"
"                     You can also use the " REDIS_CLI_AUTH_ENV " environment\n"
"                     variable to pass this password more safely\n"
"                     (if both are used, this argument takes precedence).\n"
"  --user <username>  Used to send ACL style 'AUTH username pass'. Needs -a.\n"
"  --pass <password>  Alias of -a for consistency with the new --user option.\n"
"  --askpass          Force user to input password with mask from STDIN.\n"
"                     If this argument is used, '-a' and " REDIS_CLI_AUTH_ENV "\n"
"                     environment variable will be ignored.\n"
"  -u <uri>           Server URI.\n"
"  -r <repeat>        Execute specified command N times.\n"
"  -i <interval>      When -r is used, waits <interval> seconds per command.\n"
"                     It is possible to specify sub-second times like -i 0.1.\n"
"                     This interval is also used in --scan and --stat per cycle.\n"
"                     and in --bigkeys, --memkeys, and --hotkeys per 100 cycles.\n"
"  -n <db>            Database number.\n"
"  -2                 Start session in RESP2 protocol mode.\n"
"  -3                 Start session in RESP3 protocol mode.\n"
"  -x                 Read last argument from STDIN (see example below).\n"
"  -X                 Read <tag> argument from STDIN (see example below).\n"
"  -d <delimiter>     Delimiter between response bulks for raw formatting (default: \\n).\n"
"  -D <delimiter>     Delimiter between responses for raw formatting (default: \\n).\n"
"  -c                 Enable cluster mode (follow -ASK and -MOVED redirections).\n"
"  -e                 Return exit error code when command execution fails.\n"
"  --raw              Use raw formatting for replies (default when STDOUT is\n"
"                     not a tty).\n"
"  --no-raw           Force formatted output even when STDOUT is not a tty.\n"
"  --quoted-input     Force input to be handled as quoted strings.\n"
"  --csv              Output in CSV format.\n"
"  --json             Output in JSON format (default RESP3, use -2 if you want to use with RESP2).\n"
"  --quoted-json      Same as --json, but produce ASCII-safe quoted strings, not Unicode.\n"
"  --show-pushes <yn> Whether to print RESP3 PUSH messages.  Enabled by default when\n"
"                     STDOUT is a tty but can be overridden with --show-pushes no.\n"
"  --stat             Print rolling stats about server: mem, clients, ...\n",
version,tls_usage);

    fprintf(target,
"  --latency          Enter a special mode continuously sampling latency.\n"
"                     If you use this mode in an interactive session it runs\n"
"                     forever displaying real-time stats. Otherwise if --raw or\n"
"                     --csv is specified, or if you redirect the output to a non\n"
"                     TTY, it samples the latency for 1 second (you can use\n"
"                     -i to change the interval), then produces a single output\n"
"                     and exits.\n"
"  --latency-history  Like --latency but tracking latency changes over time.\n"
"                     Default time interval is 15 sec. Change it using -i.\n"
"  --latency-dist     Shows latency as a spectrum, requires xterm 256 colors.\n"
"                     Default time interval is 1 sec. Change it using -i.\n"
"  --lru-test <keys>  Simulate a cache workload with an 80-20 distribution.\n"
"  --replica          Simulate a replica showing commands received from the master.\n"
"  --rdb <filename>   Transfer an RDB dump from remote server to local file.\n"
"                     Use filename of \"-\" to write to stdout.\n"
"  --functions-rdb <filename> Like --rdb but only get the functions (not the keys)\n"
"                     when getting the RDB dump file.\n"
"  --pipe             Transfer raw Redis protocol from stdin to server.\n"
"  --pipe-timeout <n> In --pipe mode, abort with error if after sending all data.\n"
"                     no reply is received within <n> seconds.\n"
"                     Default timeout: %d. Use 0 to wait forever.\n",
    REDIS_CLI_DEFAULT_PIPE_TIMEOUT);
"  --bigkeys          Sample Redis keys looking for keys with many elements (complexity).\n"
"  --memkeys          Sample Redis keys looking for keys consuming a lot of memory.\n"
"  --memkeys-samples <n> Sample Redis keys looking for keys consuming a lot of memory.\n"
"                     And define number of key elements to sample\n"
"  --hotkeys          Sample Redis keys looking for hot keys.\n"
"                     only works when maxmemory-policy is *lfu.\n"
"  --scan             List all keys using the SCAN command.\n"
"  --pattern <pat>    Keys pattern when using the --scan, --bigkeys or --hotkeys\n"
"                     options (default: *).\n"
"  --count <count>    Count option when using the --scan, --bigkeys or --hotkeys (default: 10).\n"
"  --quoted-pattern <pat> Same as --pattern, but the specified string can be\n"
"                         quoted, in order to pass an otherwise non binary-safe string.\n"
"  --intrinsic-latency <sec> Run a test to measure intrinsic system latency.\n"
"                     The test will run for the specified amount of seconds.\n"
"  --eval <file>      Send an EVAL command using the Lua script at <file>.\n"
"  --ldb              Used with --eval enable the Redis Lua debugger.\n"
"  --ldb-sync-mode    Like --ldb but uses the synchronous Lua debugger, in\n"
"                     this mode the server is blocked and script changes are\n"
"                     not rolled back from the server memory.\n"
"  --cluster <command> [args...] [opts...]\n"
"                     Cluster Manager command and arguments (see below).\n"
"  --verbose          Verbose mode.\n"
"  --no-auth-warning  Don't show warning message when using password on command\n"
"                     line interface.\n"
"  --help             Output this help and exit.\n"
"  --version          Output version and exit.\n"
"\n");`, version, 30)
}

func (cli *RedisCli) Run() error {
	config := &RedisCliCfg{}
	config.connInfo = &CliConnInfo{}
	config.connInfo.hostIp = "127.0.0.1"
	config.connInfo.hostPort = 6379
	config.dbNum = 0
	config.interactive = false
	config.shutdown = false
	cli.config = config

	if len(os.Args) == 0 && cli.config.eval != "" {
		cli.connect(0)
		cli.repl()
	}

	return nil
}

// connect to redis server
// flag: CCForce: The connection is performed even if there is already
// *                a connected socket.
// *    CCQuiet: Don't print errors if connection fails
func (cli *RedisCli) connect(flag CliConnectFlag) hredis.RedisStatus {
	if context == nil || flag&CCForce != 0 {
		if context != nil {
			cli.config.dbNum = 0
			cli.config.inMulti = false
		}

		// Do not use hostsocket when we got redirected in cluster mode
		if cli.config.hostSocket != "" || (!cli.config.clusterMode && cli.config.clusterReissueCommand) {
			context = hredis.RedisConnect(cli.config.connInfo.hostIp, cli.config.connInfo.hostPort)
		} else {
			context = hredis.RedisConnectUnix(cli.config.hostSocket)
		}

		if context.Err > 0 && cli.config.tls {
			if err := cli.cliSecureConnection(context); err != nil {
				fmt.Fprintf(os.Stderr, "Could not negotiate a TLS connection: %s\n", err.Error())
				context = nil
				return hredis.RedisErr
			}
		}

		if context.Err > 0 {
			if flag&CCQuiet == 0 {
				// TODO
			}
			return hredis.RedisErr
		}

		err := syscall.SetsockoptInt(context.RedisFd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set SO_KEEPALIVE: %s\n", err.Error())
		}

		// Do Auth, select the right DB, switch to RESP3 if needed
		if ok := cli.auth(context, cli.config.connInfo.user, cli.config.connInfo.auth); ok != hredis.RedisOk {
			return hredis.RedisErr
		}

		if ok := cli.selectInputDb(context); ok != hredis.RedisOk {
			return hredis.RedisErr
		}

		if ok := cli.switchProto(context); ok != hredis.RedisOk {
			return hredis.RedisErr
		}
	}

	return hredis.RedisOk
}

func (cli *RedisCli) repl() {

	var (
		history     bool
		historyFile string
	)

	//There is no need to initialize redis HELP when we are in lua debugger mode.
	// It has its own HELP and commands (COMMAND or COMMAND DOCS will fail and got nothing).
	//We will initialize the redis HELP after the Lua debugging session ended.
	if !cli.config.evalLDB && isatty.IsTerminal(os.Stdin.Fd()) {
		/* Initialize the help using the results of the COMMAND command. */
		cli.initHelp(context)
	}

	cli.config.interactive = true
	if isatty.IsTerminal(os.Stdin.Fd()) {
		historyFile = getDotfilePath(RedisCliHisFileEnv, RedisCliHisFileDefault)
		//keep in-memory history always regardless if history file can be determined
		history = true
		if historyFile != "" {
			linenoise.Line.HistoryLoad(historyFile)
		}
		cliLoadPreferences()
	}

	cli.cliRefreshPrompt()
	for {
		var prompt string
		if context != nil {
			prompt = cli.config.prompt
		} else {
			prompt = fmt.Sprintf("not connected> ")
		}
		line, err := linenoise.Line.Prompt(prompt)
		if err != nil {
			if cli.config.pubsubMode {
				cli.config.pubsubMode = false
				if cli.connect(CCQuiet) == hredis.RedisOk {
					continue
				}
			}
			break
		}

		argv, argc := cli.splitArgs(line)
		if argv == nil {
			fmt.Printf("Invalid argument(s)\n")
			if history {
				linenoise.Line.AppendHistory(line)
			}
			if historyFile != "" {
				linenoise.Line.HistorySave(historyFile)
			}
			continue
		} else if argc == 0 {
			continue
		}

		// check if we have a repeat command option and need to skip the first arg
		repeat, err := strconv.Atoi(argv[0])
		skipargs := 0

		if argc > 1 && err == nil {
			// Checking for error scenarios
			if repeat <= 0 {
				fmt.Println("Invalid redis-cli repeat command option value.")
				continue
			}
			skipargs = 1
		} else {
			repeat = 1
		}

		if strings.EqualFold(argv[0], "quit") || strings.EqualFold(argv[0], "exit") {
			os.Exit(0)
		} else if argv[0][0] == ':' {
			cliSetPreferences(argv, argc, true)
			continue
		} else if strings.EqualFold(argv[0], "restart") {
			if cli.config.eval != "" {
				cli.config.evalLDB = true
				cli.config.output = OutputRaw
				return
			} else {
				fmt.Printf("Use 'restart' only in Lua debugging mode.\n")
			}
		} else if argc == 3 && strings.EqualFold(argv[0], "connect") {
			cli.config.connInfo.hostIp = argv[1]
			cli.config.connInfo.hostPort, err = strconv.Atoi(argv[2])
			if err != nil {
				fmt.Printf("Invalid port number\n")
				return
			}
			cli.cliRefreshPrompt()
			cli.connect(CCForce)
		} else if argc == 1 && strings.EqualFold(argv[0], "clear") {
			linenoise.Line.ClearScreen()
		} else {

			startTime := time.Now()

			/* If our debugging session ended, show the EVAL final
			 * reply. */
			if cli.config.evalLDBEnd {
				cli.config.evalLDBEnd = false

			}
		}

	}
	return
}

func (cli *RedisCli) ReadReply(outputRawStrings string) {
	if cli.config.lastReply != nil {
		cli.config.lastReply = nil
	}

}

func (cli *RedisCli) splitArgs(line string) ([]string, int) {
	if cli.config.evalLDB && (strings.HasPrefix(line, "eval ") || strings.HasPrefix(line, "e ")) {
		var argv []string
		argc := 2
		elen := 0
		if line[1] == ' ' {
			elen = 2 // "e "
		} else {
			elen = 5 // "eval "
		}
		argv = append(argv, line[:elen-1])
		argv = append(argv, line[elen:])
		return argv, argc
	} else {
		argv := strings.Fields(line)
		return argv, len(argv)
	}

}

/* initHelp sets up the helpEntries array with the command and group
 * names and command descriptions obtained using the COMMAND DOCS command.
 */
// TODO: the command docs
func (cli *RedisCli) initHelp(ctx *hredis.RedisContext) {
	// Dict type for a set of strings, used to collect names of command groups

	var (
		commandTable *hredis.RedisReply
	)

	if ok := cli.connect(CCQuiet); ok == hredis.RedisErr {
		/* Can not connect to the server, but we still want to provide
		 * help, generate it only from the static cli_commands.c data instead. */
		return
	}
	commandTable = ctx.RedisCommand("COMMAND DOCS")
	if commandTable == nil || commandTable.Tp == hredis.RedisReplyError {

	}
}

// cliSecureConnection wrapper around redisSecureConnection to avoid hredis_ssl deps
func (cli *RedisCli) cliSecureConnection(ctx *hredis.RedisContext) error {
	return nil
}

/*------------------------------------------------------------------------------
 * Networking / parsing
 *--------------------------------------------------------------------------- */

// auth send AUTH command to the server
func (cli *RedisCli) auth(ctx *hredis.RedisContext, user, auth string) hredis.RedisStatus {
	var reply *hredis.RedisReply
	if auth == "" {
		return hredis.RedisOk
	}

	if user == "" {
		reply = ctx.RedisCommand("AUTH %s", auth)
	} else {
		reply = ctx.RedisCommand("AUTH %s %s", auth, user)
	}

	if reply == nil {
		fmt.Fprintf(os.Stderr, "\nI/O error\n")
		return hredis.RedisErr
	}

	result := hredis.RedisOk
	if reply.Tp == hredis.RedisReplyError {
		result = hredis.RedisErr
		fmt.Fprintf(os.Stderr, "AUTH failed: %s\n", reply.Str)
	}
	return result
}

// selectInputDb send select input_dbnum to the server
func (cli *RedisCli) selectInputDb(ctx *hredis.RedisContext) hredis.RedisStatus {
	var reply *hredis.RedisReply
	if cli.config.connInfo.inputDbNum == cli.config.dbNum {
		return hredis.RedisOk
	}

	reply = ctx.RedisCommand("SELECT %d", cli.config.connInfo.inputDbNum)
	if reply == nil {
		fmt.Fprintf(os.Stderr, "\nI/O error\n")
		return hredis.RedisErr
	}

	result := hredis.RedisOk
	if reply.Tp == hredis.RedisReplyError {
		result = hredis.RedisErr
		fmt.Fprintf(os.Stderr, "SELECT %d failed: %s\n", cli.config.connInfo.inputDbNum, reply.Str)
	} else {
		cli.config.dbNum = cli.config.connInfo.inputDbNum
		cli.cliRefreshPrompt()
	}
	return result
}

// switchProto switch to RESP3 if needed
func (cli *RedisCli) switchProto(ctx *hredis.RedisContext) hredis.RedisStatus {
	var reply *hredis.RedisReply
	if cli.config.resp3 < 0 || cli.config.resp2 > 0 {
		return hredis.RedisOk
	}

	reply = ctx.RedisCommand("HELLO 3")
	if reply == nil {
		fmt.Fprintf(os.Stderr, "\nI/O error\n")
		return hredis.RedisErr
	}

	result := hredis.RedisOk
	if reply.Tp == hredis.RedisReplyError {
		fmt.Fprintf(os.Stderr, "HELLO 3 failed: %s\n", reply.Str)
		if cli.config.resp3 == 1 {
			result = hredis.RedisErr
		} else if cli.config.resp3 == 2 {
			result = hredis.RedisOk
		}
	}

	// Retrieve server version string for later use
	for i := 0; i < reply.Elements; i += 2 {
		if reply.Element[i].Str == "server" {
			cli.config.serverVersion = reply.Element[i+1].Str
			break
		}
	}

	cli.config.currentResp3 = true
	return result
}

func (cli *RedisCli) cliRefreshPrompt() {
	if cli.config.evalLDB {
		return
	}

	var prompt string

	if cli.config.hostSocket != "" {
		prompt = fmt.Sprintf("redis %s", cli.config.hostSocket)
	} else {
		prompt = fmt.Sprintf("redis://%s:%d", cli.config.connInfo.hostIp, cli.config.connInfo.hostPort)
	}

	// Add [dbnum] if needed
	if cli.config.dbNum != 0 {
		prompt = fmt.Sprintf("%s[%d]", prompt, cli.config.dbNum)
	}

	// Add TX if in transaction state
	if cli.config.inMulti {
		prompt = fmt.Sprintf("%s[TX]", prompt)
	}

	if cli.config.pubsubMode {
		prompt = fmt.Sprintf("%s[pubsub]", prompt)
	}

	prompt = fmt.Sprintf("%s> ", prompt)
	cli.config.prompt = prompt
}

/*------------------------------------------------------------------------------
 * Help functions
 *--------------------------------------------------------------------------- */

type CliHelpType uint8

const (
	CliHelpCommand CliHelpType = iota
	CliHelpGroup
)

type CliHelpEntry struct {
	tp   CliHelpType
	argc int
	argv []string
	full string
	docs commandDocs
}

func getDotfilePath(envOverride, dotFilename string) string {
	var dotPath string

	path := os.Getenv(envOverride)
	if path != "" {
		if path == "/dev/null" {
			return ""
		}
		dotPath = path
	} else {
		home := os.Getenv("HOME")
		if home != "" {
			dotPath = fmt.Sprintf("%s/%s", home, dotFilename)
		}
	}
	return dotPath
}

// cliLoadPreferences load the ~/.redisclirc file
func cliLoadPreferences() {
	rcFile := getDotfilePath(RedisCliRCFileEnv, RedisCliRCFileDefault)
	if rcFile == "" {
		return
	}

	// TODO
}

// cliSetPreferences set the ~/.redisclirc file
func cliSetPreferences(argv []string, argc int, interactive bool) {
	// TODO
}
