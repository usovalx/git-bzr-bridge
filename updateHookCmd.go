package main

import (
	"github.com/usovalx/git-bzr-bridge/bzr"
	"github.com/usovalx/git-bzr-bridge/git"

	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

	// export git -> import bzr & push it
	exportGitImportBzrAndPush(fs.Arg(2), gitBranch, bzrBranch, url)
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

func exportGitImportBzrAndPush(
	gitRev, gitBranch, bzrBranch, url string) {

	// FIXME: use locks

	tmpGitBranch := "__git_import/" + gitBranch
	tmpBzrBranch := filepath.FromSlash(path.Join(bzrRepo, tmpGitBranch))

	// create all temp files we will need later
	tmpBzrMarks, err := ioutil.TempFile(tmpDir, "bzr_marks")
	must(err)
	defer os.Remove(tmpBzrMarks.Name())
	tmpBzrMarks.Close()
	tmpGitMarks, err := ioutil.TempFile(tmpDir, "git_marks")
	must(err)
	defer os.Remove(tmpGitMarks.Name())
	tmpGitMarks.Close()

	// create a temp branch in git
	must(git.NewBranch(tmpGitBranch, gitRev))
	defer git.RemoveBranch(tmpGitBranch)

	// export data into bzr
	log.Info("Exporting data from git")
	defer os.RemoveAll(tmpBzrBranch)
	exportSize, err := RunPipe(
		git.Export(tmpGitBranch, gitMarks, tmpGitMarks.Name()),
		bzr.Import(bzrRepo, bzrMarks, tmpBzrMarks.Name()))

	if exportSize == 0 {
		log.Info("Empty export. Creating bzr branch using marks")
		b, err := loadMarks(bzrMarks)
		must(err)
		g, err := loadMarks(gitMarks)
		must(err)
		mark, ok := g.byRev[gitRev]
		if !ok {
			log.Panicf("Can't find revision %q in the git marks file", gitRev)
		}
		brev, ok := b.byMark[mark]
		if !ok {
			log.Panicf("Can't find mark %d in bzr marks file", mark)
		}
		must(bzr.NewBranch(tmpBzrBranch, brev))
	}

	log.Info("Pushing into bzr")
	must(bzr.Push(tmpBzrBranch, url))

	log.Info("Finalizing")
	must(bzr.PullOverwrite(tmpBzrBranch, bzrBranch))
	if exportSize != 0 {
		must(os.Rename(tmpBzrMarks.Name(), bzrMarks))
		must(os.Rename(tmpGitMarks.Name(), gitMarks))
	}
}
