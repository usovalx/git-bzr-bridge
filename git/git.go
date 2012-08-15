package git

import (
	l "gitbridge/log"
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
	return git("init", "--bare", path).Run()
}

// Prepare exec.Cmd to run git with specified arguments
func git(args ...string) *exec.Cmd {
	a := append(conf.GitCommand, args...)
	log.Debugf("Running %q", strings.Join(a, " "))
	return exec.Command(a[0], a[1:]...)
}
