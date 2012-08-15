package main

import (
	"gitbridge/bzr"
	"gitbridge/git"

	"flag"
	"fmt"
	"os"
)

func initCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	fs.SetOutput(os.Stdout)
	fs.Usage = func() { initUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}

	repoPath := fs.Arg(0)
	log.Debug("Creating ", repoPath)
	must(os.Mkdir(repoPath, 0777))
	must(os.Chdir(repoPath))

	log.Debug("Initializing git repo")
	must(git.InitRepo("."))
	log.Debug("Initializing bzr repo")
	must(bzr.InitRepo("bzr"))

	os.Exit(0)
}

func initUsage(fs *flag.FlagSet) {
	fmt.Println("usage: gitbridge init [-h] <path>")
	fmt.Println("\nflags:")
	fs.PrintDefaults()

	fmt.Print(`
init will initialize a new repository at <path>
`)
}
