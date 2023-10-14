package cmd

import (
	"github.com/fzft/go-mock-redis/node"
)

// cliCommandArg syntax spec for a command argument.
type cliCommandArg struct {
	name        string
	tp          node.CommandArgType
	token       string
	since       string
	flags       int
	numArgs     int
	subArgs     []*cliCommandArg
	displayText string

	// For use at runtime.
	// Fields used to keep track of input word matches for command-line hints.
	matched      int
	matchedToken int
	matchedName  int
	matchedAll   int
}

// commandDocs documentation info used for help command.
type commandDocs struct {
	name    string
	summary string
	group   string
	since   string
	numArgs int
}
