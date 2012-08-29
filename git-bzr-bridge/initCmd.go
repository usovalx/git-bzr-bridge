package main

import (
	"github.com/usovalx/git-bzr-bridge/bzr"
	"github.com/usovalx/git-bzr-bridge/git"

	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func initCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
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
	// it's ok if given folder already exists
	// example: init .
	os.Mkdir(repoPath, 0777)
	must(os.Chdir(repoPath))

	// first try to initialize bzr repo, so that it will fail early in case of any issues
	log.Debug("Initializing bzr repo")
	must(bzr.InitRepo(bzrRepo))
	log.Debug("Initializing git repo")
	must(git.InitRepo("."))
	log.Debug("Creating branch config")
	must(ioutil.WriteFile(branchConfigName, []byte("[]"), 0666))
	log.Debug("Creating marks files")
	must(ioutil.WriteFile(bzrMarks, []byte{}, 0666))
	must(ioutil.WriteFile(gitMarks, []byte{}, 0666))
	log.Debug("Creating temp dir")
	must(os.Mkdir(tmpDir, 0777))
}

func initUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge init [-h] <path>")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
init will initialize a new repository at <path>
`)
}
