package cmd

import (
	"fmt"
	"github.com/fzft/go-mock-redis/deps/hredis"
	"io"
	"os"
	"strconv"
)

var RedisVersion = "255.255.255"
var RedisVersionNum = 0x00ffffff

type CliConnectFlag int

var context *hredis.RedisContext

const (
	CCForce CliConnectFlag = 1 << iota // Re-connect if already connected.
	CCQuiet                            // Don't show non-error messages.
)

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
	tls                   int
	repeat                int64
	dbNum                 int
	interactive           bool
	shutdown              bool
	inMulti               bool
	eval                  string
	clusterMode           bool
	clusterReissueCommand bool
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

	if len(os.Args) == 1 {

		return nil
	}

	return nil
}

// connect to redis server
// flag: CCForce: The connection is performed even if there is already
// *                a connected socket.
// *    CCQuiet: Don't print errors if connection fails
func (cli *RedisCli) connect(flag CliConnectFlag) error {
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
	}
	return
}

func (cli *RedisCli) repl() error {
	return nil
}
