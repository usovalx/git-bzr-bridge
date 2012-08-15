package bzr

import (
	l "gitbridge/log"
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
	if _, err := Output("help"); err != nil {
		log.Error("Bazaar is not runnable: ", err)
		return false
	}

	// if fast-export is available
	log.Info("Testing whether Bazaar has fast-export plugin")
	if out, err := Output("fast-export", "--usage"); err != nil {
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

// Run bzr and grab its output
func Output(bzrArgs ...string) ([]byte, error) {
	args := append(conf.BzrCommand, bzrArgs...)
	log.Debugf("Running %q", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.Output()
	return out, err
}
