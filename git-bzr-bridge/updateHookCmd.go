package main

import (
	"github.com/usovalx/git-bzr-bridge/bzr"
	"github.com/usovalx/git-bzr-bridge/git"

	"flag"
	"fmt"
	"os"
	"strings"
)

func updateHookCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("update-hook", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	fs.Usage = func() { updateHookUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if fs.NArg() != 3 || fs.Arg(0) == "" || fs.Arg(1) == "" || fs.Arg(2) == "" {
		fs.Usage()
		os.Exit(2)
	}

	const emptyRef = "0000000000000000000000000000000000000000"

	// FIXME: should it make an exception for tags?
	if fs.Arg(1) == emptyRef {
		panic(fmt.Errorf("Creation of new references is not supported"))
	}

	if fs.Arg(2) == emptyRef {
		panic(fmt.Errorf("Deletion of reference is not supported"))
	}

	// FIXME: locking

	// trim "/refs/heads/" prefix from git reference
	const prefix = "refs/heads/"
	var gitBranch string
	if !strings.HasPrefix(fs.Arg(0), prefix) {
		log.Errorf("Unexpected reference name %q", fs.Arg(0))
		os.Exit(1)
	}
	gitBranch = fs.Arg(0)[len(prefix):]

	// find corresponding bzr branch
	branchConfig, err := loadBranchConfig()
	must(err)
	var bzrBranch, url string
	if c, ok := branchConfig.byGitName[gitBranch]; !ok {
		log.Errorf("Unknown branch %q", gitBranch)
		os.Exit(1)
	} else {
		bzrBranch = c.Bzr
		url = c.Url
	}

	// now let's try to update bazaar branch to reduce the possibility of diverged branches
	updated := cloneAndExportBzrImportGit(
		url,
		checkIfBranchUpdated(bzrBranch),
		// FIXME: move this bit into function
		func(marksUpdated bool, tmpGitMarks, tmpBzrMarks, tmpGitBranch, tmpBzrBranch string) {
			must(bzr.PullOverwrite(tmpBzrBranch, bzrBranch))
			must(git.RenameBranch(tmpGitBranch, gitBranch))
			if marksUpdated {
				must(os.Rename(tmpBzrMarks, bzrMarks))
				must(os.Rename(tmpGitMarks, gitMarks))
			}
		})
	if updated {
		log.Error("These branches have diverged")
		os.Exit(1)
	}

	if !checkFastForward(fs.Arg(1), fs.Arg(2)) {
		log.Error("Not fast-forward push")
		os.Exit(1)
	}

	// TODO: export git -> import bzr & push it
}

func updateHookUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge update-hook [-h] <ref name> <old obj> <new obj>")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
update 
`)
}

func checkFastForward(old, new string) bool {
	r, err := git.LeftRevList(old, new)
	must(err)

	return len(r) == 0
}
