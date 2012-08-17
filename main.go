package main

import (
	l "gitbridge/log"

	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var log = l.New("gitbridge")

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

type marks struct {
	byRev  map[string]int
	byMark map[int]string
}

func main() {
	// global command-line flags
	fs := flag.NewFlagSet("gitbridge", flag.ExitOnError)
	var verbose = fs.Bool("v", false, "more verbose logging")
	var help = fs.Bool("h", false, "show usage message and options")
	var wd = fs.String("C", "", "change to this directory before doing anything else")
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
		if err := os.Chdir(*wd); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	rand.Seed(time.Now().UnixNano())

	// choose subcommand and run it
	if fs.NArg() > 0 {
		if cmd, ok := commands[fs.Arg(0)]; ok {
			defer func() {
				if e := recover(); e != nil {
					os.Exit(1)
				} else {
					os.Exit(0)
				}
			}()
			cmd.cmd(fs.Args()[1:])
			os.Exit(0)
		}
	}

	// failed to run subcommand -- show usage and exit with error
	fs.Usage()
	os.Exit(2)
}

func showUsage(fs *flag.FlagSet) {
	fmt.Println("usage: gitbridge [flags] <command> [<args>]")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
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
		panic(err)
	}
}

func tempBranchName() string {
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

func loadMarks(path string) (*marks, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bf := bufio.NewReader(f)

	m := new(marks)
	m.byMark = make(map[int]string)
	m.byRev = make(map[string]int)

	for {
		line, err := bf.ReadString('\n')
		s := strings.Split(line, " ")
		if len(s) == 2 {
			rev := strings.TrimSpace(s[1])
			mark, err := strconv.Atoi(s[0][1:])
			if err != nil {
				log.Errorf("%s: invalid mark: %s", path, line)
				continue
			}
			m.byRev[rev] = mark
			m.byMark[mark] = rev
		}
		if err == io.EOF {
			return m, nil
		}
		if err != nil {
			return nil, err
		}
	}
	panic("unreachable")
}
