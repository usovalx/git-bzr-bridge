package main

import (
	"flag"
	"os"
	"fmt"
	)

func updateCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	updateAll := fs.Bool("a", false, "update all branches")
	fs.Usage = func() { updateUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if (*updateAll && fs.NArg() != 0) || (!*updateAll && fs.NArg() == 0) {
		fs.Usage()
		os.Exit(2)
	}
}

func updateUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge update [-h] [-a] [<branch>]")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
update specified <branch> or all known branches (if -a is specified). Essentially
will pull new revisions from bzr and import them into git.
`)
}