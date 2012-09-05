package git

import (
	l "github.com/usovalx/git-bzr-bridge/log"

	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Config options
type Config struct {
	GitCommand []string
}

// Active config
var conf = Config{
	GitCommand: []string{"git"},
}

var log = l.New("git")

// Set config options for the package
func SetConfig(c Config) {
	if len(c.GitCommand) < 1 {
		log.Panicf("Invalid command line for git: too short: %#v", c.GitCommand)
	}
}

// Test whether git is runnable with current config
func TestInstall() bool {
	log.Debugf("Current git config: %+v", conf)

	// check if bzr is runnable
	log.Info("Testing whether Git is runnnable")
	if err := git("help").Run(); err != nil {
		log.Error("Git is not runnable: ", err)
		return false
	}

	log.Info("Testing whether Git has fast-export")
	if out, err := git("fast-export", "--help").Output(); err != nil {
		log.Error("fast-export isn't working: ", err)
		return false
	} else {
		req := []string{"-M", "-C", "--export-marks", "--import-marks"}
		usage := string(out)
		for _, s := range req {
			if !strings.Contains(usage, s) {
				log.Errorf("fast-export doesn't support %q", s)
				return false
			}
		}
	}

	log.Info("Git installation seems ok")
	return true
}

// Initialize git repo at the given path
func InitRepo(path string) error {
	return run(git("init", "--bare", path))
}

func Import(inMarks, outMarks string) *exec.Cmd {
	flags := []string{
		"fast-import", "--force",
		"--import-marks=" + inMarks,
		"--export-marks=" + outMarks}
	if l.MinLogLevel < l.DEBUG {
		flags = append(flags, "--quiet")
	}
	c := git(flags...)
	c.Stdout = os.Stdout
	return c
}

func Export(branch, inMarks, outMarks string) *exec.Cmd {
	flags := []string{
		"fast-export", "-M", "-C",
		"--import-marks=" + inMarks,
		"--export-marks=" + outMarks,
		branch}
	return git(flags...)
}

func RenameBranch(from, to string) error {
	return run(git("branch", "-M", from, to))
}

func RemoveBranch(name string) error {
	return run(git("branch", "-D", name))
}

func NewBranch(name, rev string) error {
	return run(git("branch", name, rev))
}

func LeftRevList(old, new string) ([]byte, error) {
	return git("rev-list", "--left-only", old+"..."+new).Output()
}

// Prepare exec.Cmd to run git with specified arguments
func git(args ...string) *exec.Cmd {
	a := append(conf.GitCommand, args...)
	log.Debugf("Running %q", strings.Join(a, " "))
	c := exec.Command(a[0], a[1:]...)
	c.Stderr = os.Stderr
	return c
}

func run(c *exec.Cmd) error {
	r := c.Run()
	if r != nil {
		return fmt.Errorf("%s %s: %s", c.Path, c.Args[0], r)
	}
	return nil
}
