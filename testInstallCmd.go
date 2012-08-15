package main

import (
	"gitbridge/bzr"
	"gitbridge/git"

	"flag"
	"fmt"
	"os"
)

func testInstallCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("test-install", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	fs.SetOutput(os.Stdout)
	fs.Usage = func() { testInstallUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}
	if fs.NArg() != 0 {
		fs.Usage()
		os.Exit(2)
	}

	if bzr.TestInstall() && git.TestInstall() {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func testInstallUsage(fs *flag.FlagSet) {
	fmt.Println("usage: gitbridge test-install [-h]")
	fmt.Println("\nflags:")
	fs.PrintDefaults()

	fmt.Print(`
test-install does a basic check of configuration and tools. It will
try to run bzr and git and verify that all required plugins are
installed and working.
`)
}
