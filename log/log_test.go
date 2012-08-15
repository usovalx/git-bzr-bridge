package log

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

var stderr, stdout *bytes.Buffer

func prep() {
	stderr = new(bytes.Buffer)
	stdout = new(bytes.Buffer)
	std = log.New(stdout, "", log.LstdFlags)
	err = log.New(stderr, "", log.LstdFlags)
}

func TestNormalLevels(t *testing.T) {
	for minLevel := DEBUG; minLevel <= NONE; minLevel++ {
		for level := DEBUG; level < PANIC; level++ {
			prep()
			l := New("testLogger")
			MinLogLevel = minLevel
			l.log(level, 1, "testMessage")
			str := string(stdout.Bytes())

			if level >= minLevel {
				if !(strings.Contains(str, "testLogger") && strings.Contains(str, "testMessage")) {
					t.Errorf("Message not logged at level = %s and minLevel = %s", level, minLevel)
				}
			} else {
				if str != "" {
					t.Errorf("Message is logged at level = %s and minLevel = %s", level, minLevel)
				}
			}

			if string(stderr.Bytes()) != "" {
				t.Errorf("Unexpected message in stderr at level = %s and minLevel = %s",
					level, minLevel)
			}
		}
	}
}

func TestFailPanicLevels(t *testing.T) {
	for minLevel := DEBUG; minLevel <= NONE; minLevel++ {
		prep()
		level := PANIC
		l := New("testLogger")
		MinLogLevel = minLevel
		l.log(level, 1, "testMessage")
		str := string(stderr.Bytes())

		if !(strings.Contains(str, "testLogger") && strings.Contains(str, "testMessage")) {
			t.Errorf("Message not logged at level = %s and minLevel = %s", level, minLevel)
		}

		if string(stdout.Bytes()) != "" {
			t.Errorf("Unexpeded message in stdout at level = %s and minLevel = %s",
				level, minLevel)
		}
	}
}
