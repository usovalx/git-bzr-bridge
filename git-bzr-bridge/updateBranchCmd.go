package main

import (
	"github.com/usovalx/git-bzr-bridge/bzr"
	"github.com/usovalx/git-bzr-bridge/git"

	"flag"
	"fmt"
	"os"
)

func updateCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	updateAll := fs.Bool("a", false, "update all branches")
	fs.Usage = func() { updateUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if (*updateAll && fs.NArg() != 0) || (!*updateAll && fs.NArg() == 0) {
		fs.Usage()
		os.Exit(2)
	}

	branchConfig, err := loadBranchConfig()
	must(err)

	var toUpdate []string
	if *updateAll {
		for _, v := range branchConfig.branches {
			toUpdate = append(toUpdate, v.Git)
		}
	} else {
		toUpdate = fs.Args()
	}

	errors := false
	for _, branch := range toUpdate {
		if v, ok := branchConfig.byGitName[branch]; ok {
			if e := doUpdateBranch(v.Git, v.Bzr, v.Url); e != nil {
				log.Error(e)
				errors = true
			}
		} else {
			log.Errorf("Branch %q isn't valid", branch)
			errors = true
		}
	}

	if errors {
		panic(fmt.Errorf("Some branches failed to update"))
	}
}

func updateUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge update [-h] [-a] [<branch>]")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
update specified <branch> or all known branches (if -a is specified). Essentially
will pull new revisions from bzr and import them into git.

If multiple branches are specified it will try to updates them all. When updating
multiple branches, update won't stop early on errors and will try to update all
requested branches.
`)
}

func doUpdateBranch(gitBranch, bzrBranch, url string) (err error) {
	log.Infof("Updating %q from %q", gitBranch, url)

	// capture any panics and convert them into errors
	defer func() {
		if e := recover(); e != nil {
			if e, ok := e.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%s", e)
			}
		}
	}()

	cloneAndExportBzrImportGit(url,
		func(marksUpdated bool, tmpGitMarks, tmpBzrMarks, tmpGitBranch, tmpBzrBranch string) {
			must(bzr.PullOverwrite(tmpBzrBranch, bzrBranch))
			must(git.RenameBranch(tmpGitBranch, gitBranch))
			if marksUpdated {
				must(os.Rename(tmpBzrMarks, bzrMarks))
				must(os.Rename(tmpGitMarks, gitMarks))
			}
		})
	return nil
}
