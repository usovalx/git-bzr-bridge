package main

import (
	l "gitbridge/log"

	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"time"
)

var log = l.New("gitbridge.main")

// some constants
const bzrRepo = "bzr"
const branchConfigName = "gitbridge_branches.cfg"
const bzrMarks = "gitbridge_bzr.marks"
const gitMarks = "gitbridge_git.marks"
const tmpDir = "gitbridge_tmp"

type commandInfo struct {
	cmd         func([]string)
	description string
}

var commands = map[string]commandInfo{
	"init":         {initCmd, "create a new repository"},
	"import":       {importCmd, "import new bzr branch"},
	"test-install": {testInstallCmd, "basic check of the setup"},
}

type branchInfo struct {
	Url, Bzr, Git string
}

type branchConfig struct {
	branches  []*branchInfo
	byBzrName map[string]*branchInfo
	byGitName map[string]*branchInfo
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

	rand.Seed(time.Now().UnixNano())

	// choose subcommand and run it
	if fs.NArg() > 0 {
		if cmd, ok := commands[fs.Arg(0)]; ok {
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
	cmds := make([]string, 0, len(commands))
	for k := range commands {
		cmds = append(cmds, k)
	}
	sort.Strings(cmds)

	fmt.Println("\ncommands:")
	for _, k := range cmds {
		fmt.Printf("  %-12s  %s\n", k, commands[k].description)
	}

	fmt.Println("\nRun 'gitbridge <command> -h' to get usage message for the <command>")
}

func must(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func tempGitBranch() string {
	// FIXME: should it also check that branch doesn't exists yet?
	return fmt.Sprintf("__bzr_import_%d_%d", os.Getpid(), rand.Uint32())
}

func loadBranchConfig() (*branchConfig, error) {
	// FIXME: file locking

	// read file
	data, ferr := ioutil.ReadFile(branchConfigName)
	if ferr != nil {
		return nil, ferr
	}

	arr := make([]*branchInfo, 0, 10)
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}

	// fill in cross-maps of the branchConfig and validate it on the way
	c := new(branchConfig)
	c.branches = arr
	c.byBzrName = make(map[string]*branchInfo, len(c.branches)+1)
	c.byGitName = make(map[string]*branchInfo, len(c.branches)+1)
	err := func(i int, s string) error {
		return fmt.Errorf("%s: %s in config entry %d", branchConfigName, s, i)
	}
	for i, b := range c.branches {
		if b.Url == "" {
			return nil, err(i, "empty url")
		}
		if b.Bzr == "" {
			return nil, err(i, "empty bzr branch name")
		}
		if b.Git == "" {
			return nil, err(i, "empty git branch name")
		}
		if _, ok := c.byBzrName[b.Bzr]; ok {
			return nil, err(i, "duplicate bzr branch name")
		}
		if _, ok := c.byGitName[b.Git]; ok {
			return nil, err(i, "duplicate git branch name")
		}
		c.byBzrName[b.Bzr] = b
		c.byGitName[b.Git] = b
	}
	return c, nil
}

func addBranchToConfig(url, bzrName, gitName string) error {
	// FIXME: file locking
	data, ferr := ioutil.ReadFile(branchConfigName)
	if ferr != nil {
		return ferr
	}

	arr := make([]*branchInfo, 0, 10)
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	arr = append(arr, &branchInfo{Url: url, Bzr: bzrName, Git: gitName})
	data, err := json.MarshalIndent(arr, "", " ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(branchConfigName, data, 0777); err != nil {
		return err
	}
	return nil
}

func moveFile(from, to string) error {
	return os.Rename(from, to)
}
