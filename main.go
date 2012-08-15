package main

import (
	"gitbridge/bzr"
	"gitbridge/git"
	l "gitbridge/log"
)

var log = l.New("gitbridge.main")

func main() {
	l.MinLogLevel = l.DEBUG
	bzr.TestInstall()
	git.TestInstall()
}
