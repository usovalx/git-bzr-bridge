package main

import (
	l "github.com/usovalx/git-bzr-bridge/log"

	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

var log = l.New("git-bzr-bridge")

// some constants
const bzrRepo = "bzr"
const branchConfigName = "git-bzr-bridge-branches.cfg"
const bzrMarks = "git-bzr-bridge-bzr.marks"
const gitMarks = "git-bzr-bridge-git.marks"
const tmpDir = "git-bzr-bridge-tmp"

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
	defer func() {
		if e := recover(); e != nil {
			log.Error(e)
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}()

	// global command-line flags
	fs := flag.NewFlagSet("git-bzr-bridge", flag.ExitOnError)
	var verbose = fs.Bool("v", false, "more verbose logging")
	var debug = fs.Bool("d", false, "debug logging")
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
	if *debug {
		l.MinLogLevel = l.SPAM
	}

	if *wd != "" {
		must(os.Chdir(*wd))
	}

	rand.Seed(time.Now().UnixNano())

	// choose subcommand and run it
	if fs.NArg() > 0 {
		if cmd, ok := commands[fs.Arg(0)]; ok {
			cmd.cmd(fs.Args()[1:])
			os.Exit(0)
		}
	}

	// failed to run subcommand -- show usage and exit with error
	fs.Usage()
	os.Exit(2)
}

func showUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge [flags] <command> [<args>]")
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

	fmt.Println("\nRun 'git-bzr-bridge <command> -h' to get usage message for the <command>")
}

func must(err error) {
	if err != nil {
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

type CountReader struct {
	read uint64
	r io.ReadCloser
}

func (r *CountReader) Read(b []byte) (int, error) {
	n, e := r.r.Read(b)
	r.read += uint64(n)
	return n, e
}

func (r *CountReader) Close() error {
	return r.r.Close()
}

func (r *CountReader) NData() uint64 {
	return r.read
}

func NewCountReader(r io.ReadCloser) *CountReader {
	return &CountReader{0, r}
}

func RunPipe(src, dst *exec.Cmd) (uint64, error) {
	if src.Stdout != nil {
		return 0, fmt.Errorf("RunPipe: stdout already set on source")
	}
	if dst.Stdin != nil {
		return 0, fmt.Errorf("RunPipe: stdin already set on dest")
	}
	log.Spamf("RunPipe: src=%q  dst=%q", src.Path, dst.Path)

	p, err := src.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cr := NewCountReader(p)
	dst.Stdin = cr

	// now run them both
	log.Spam("RunPipe: starting src")
	err = src.Start()
	if err != nil {
		log.Spam("RunPipe: failed")
		return 0, err
	}

	log.Spam("RunPipe: starting dst")
	err = dst.Start()
	if err != nil {
		log.Spam("RunPipe: failed")
		p.Close()
		log.Spam("RunPipe: waiting for src to die")
		src.Wait()
		return 0, err
	}

	log.Spam("RunPipe: waiting for src to finish")
	err = src.Wait()
	log.Spam("RunPipe: waiting for dst to finish")
	err1 := dst.Wait()
	if err != nil {
		return cr.NData(), err
	}
	return cr.NData(), err1
}
