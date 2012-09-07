package bzr

import (
	l "github.com/usovalx/git-bzr-bridge/log"

	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Config options
type Config struct {
	BzrCommand []string
}

// Active config
var conf = Config{
	BzrCommand: []string{"bzr"},
}

var log = l.New("bzr")

// Set config options for the package
func SetConfig(c Config) {
	if len(c.BzrCommand) < 1 {
		log.Panicf("Invalid command line for bzr: too short: %+v", c.BzrCommand)
	}
}

// Test whether bzr is runnable with current config and has correct version of
// fast-export plugin
func TestInstall() bool {
	log.Debugf("Current bzr config: %+v", conf)

	// check if bzr is runnable
	log.Info("Testing whether Bazaar is runnnable")
	if err := bzr("help").Run(); err != nil {
		log.Error("Bazaar is not runnable: ", err)
		return false
	}

	// if fast-export is available
	log.Info("Testing whether Bazaar has fast-export plugin")
	if out, err := bzr("fast-export", "--usage").Output(); err != nil {
		log.Error("fast-export plugin isn't available: ", err)
		return false
	} else {
		// if fast-export supports all required 
		req := []string{"--plain", "--import-marks", "--export-marks", "--no-tags"}
		usage := string(out)
		for _, s := range req {
			if !strings.Contains(usage, s) {
				log.Errorf("fast-export doesn't support %q", s)
				return false
			}
		}
	}

	log.Info("Bazaar installation seems ok")
	return true
}

// Initialize bzr repo at the given path
func InitRepo(path string) error {
	return run(bzr("init-repo", "--no-trees", path))
}

func Clone(url, path string) error {
	flags := []string{"branch", "--no-tree"}
	if l.MinLogLevel > l.DEBUG {
		flags = append(flags, "--quiet")
	}
	return run(bzr(append(flags, url, path)...))
}

func Import(repoDir, inMarks, outMarks string) *exec.Cmd {
	flags := []string{
		"fast-import",
		"--import-marks", inMarks,
		"--export-marks", outMarks,
		"-", repoDir}
	if l.MinLogLevel > l.DEBUG {
		flags = append(flags, "--quiet")
	}
	c := bzr(flags...)
	c.Stdout = os.Stdout
	return c
}

func Export(path, gitBranch, inMarks, outMarks string) *exec.Cmd {
	flags := []string{
		"fast-export",
		"--plain", "--no-tags",
		"--import-marks", inMarks,
		"--export-marks", outMarks,
		"--git-branch", gitBranch,
		path, "-"}
	if l.MinLogLevel > l.DEBUG {
		flags = append(flags, "--quiet")
	}
	return bzr(flags...)
}

func Tip(path string) (string, error) {
	if out, err := bzr("revision-info", "-d", path).Output(); err != nil {
		return "", err
	} else {
		s := strings.Split(string(out), " ")
		if len(s) != 2 {
			return "", fmt.Errorf("bzr revision-info: invalid output %q", string(out))
		}
		return strings.TrimSpace(string(s[1])), nil
	}
	panic("unreachable")
}

func PullOverwrite(from, to string) error {
	return run(bzr("pull", "--overwrite", "-d", to, from))
}

func NewBranch(branch, rev string) error {
	err := run(bzr("init", "--create-prefix", branch))
	if err != nil {
		return err
	}
	return run(bzr("pull", "-d", branch, "-r", "revid:"+rev, branch))
}

func Push(branch, url string) error {
	return run(bzr("push", "-d", branch, url))
}

// Prepare exec.Cmd to run bzr with specified arguments
func bzr(args ...string) *exec.Cmd {
	a := append(conf.BzrCommand, args...)
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
