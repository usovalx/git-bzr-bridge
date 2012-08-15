package main

import (
	l "gitbridge/log"

	"flag"
	"fmt"
	"os"
	"sort"
)

var log = l.New("gitbridge.main")

type subCommand struct {
	cmd         func([]string)
	description string
}

var subCommands = map[string]subCommand{
	"init":         {initCmd, "create a new repository"},
	"test-install": {testInstallCmd, "basic check of the setup"},
}

func main() {
	// global command-line flags
	fs := flag.NewFlagSet("gitbridge", flag.ExitOnError)
	var verbose = fs.Bool("v", false, "more verbose logging")
	var help = fs.Bool("h", false, "show usage message and options")
	var wd = fs.String("C", "", "change to this directory before doing anything else")
	fs.SetOutput(os.Stdout)
	fs.Usage = func() { showUsage(fs) }
	fs.Parse(os.Args[1:])

	if *help {
		fs.Usage()
		os.Exit(0)
	}
	if *verbose {
		l.MinLogLevel = l.DEBUG
	}
	if *wd != "" {
		must(os.Chdir(*wd))
	}

	// choose subcommand and run it
	if fs.NArg() > 0 {
		if cmd, ok := subCommands[fs.Arg(0)]; ok {
			cmd.cmd(fs.Args()[1:])
			panic("subcommands shouldn't return")
		}
	}

	// failed to run subcommand -- show usage and exit with error
	fs.Usage()
	os.Exit(2)
}

func showUsage(fs *flag.FlagSet) {
	fmt.Println("usage: gitbridge [flags] <command> [<args>]")

	fmt.Println("\nflags:")
	fs.PrintDefaults()

	// get a list of all commands and sort them
	commands := make([]string, 0, len(subCommands))
	for k, _ := range subCommands {
		commands = append(commands, k)
	}
	sort.Strings(commands)

	fmt.Println("\ncommands:")
	for _, k := range commands {
		fmt.Printf("  %-12s  %s\n", k, subCommands[k].description)
	}

	fmt.Println("\nRun 'gitbridge <command> -h' to get usage message for the <command>")
}

func must(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
