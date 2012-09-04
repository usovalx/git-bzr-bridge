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
)

func importCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	b := fs.String("b", "", "git branch name")
	fs.Usage = func() { importUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if fs.NArg() != 2 || fs.Arg(0) == "" || fs.Arg(1) == "" {
		fs.Usage()
		os.Exit(2)
	}

	// FIXME: check we are in the correct directory

	// construct correct bzr & git branch names
	url := fs.Arg(0)
	bzrBranch := filepath.FromSlash(path.Join(bzrRepo, fs.Arg(1)))
	gitBranch := *b
	if gitBranch == "" {
		gitBranch = fs.Arg(1)
	}

	// load branch config and check that new branch names don't clash
	c, err := loadBranchConfig()
	must(err)
	if c.byBzrName[bzrBranch] != nil || c.byGitName[gitBranch] != nil {
		log.Error("Requested branch names clash with existing ones")
		os.Exit(1)
	}

	cloneAndExportBzrImportGit(
		url,
		func(_ string) bool { return true },
		func(marksUpdated bool, tmpGitMarks, tmpBzrMarks, tmpGitBranch, tmpBzrBranch string) {
			// finilize transaction
			// correct ordering of actions is important here
			// while we can live with stale temporary branches and/or files
			// and easily clean them up manually later it is extremely
			// important that we keep marks files in sync.
			must(os.MkdirAll(filepath.Dir(bzrBranch), 0777))
			must(os.Rename(tmpBzrBranch, bzrBranch))
			must(git.RenameBranch(tmpGitBranch, gitBranch))
			must(addBranchToConfig(url, bzrBranch, gitBranch))
			// no need to update marks if no new revisions were exported
			if marksUpdated {
				must(os.Rename(tmpBzrMarks, bzrMarks))
				must(os.Rename(tmpGitMarks, gitMarks))
			}
		})
}

func cloneAndExportBzrImportGit(
	url string,
	shouldExport func(tmpBzrBranch string) bool,
	finalizer func(marksUpdated bool, tmpGitMarks, tmpBzrMarks, tmpGitBranch, tmpBzrBranch string)) bool {

	tmpGitBranch := tempBranchName()
	tmpBzrBranch := filepath.FromSlash(path.Join(bzrRepo, tmpGitBranch))

	// FIXME: use locks

	// Create all temporary files we will use later
	tmpBzrMarks, err := ioutil.TempFile(tmpDir, "bzr_marks")
	must(err)
	defer os.Remove(tmpBzrMarks.Name())
	tmpBzrMarks.Close()
	tmpGitMarks, err := ioutil.TempFile(tmpDir, "git_marks")
	must(err)
	defer os.Remove(tmpGitMarks.Name())
	tmpGitMarks.Close()

	log.Info("Cloning bzr branch")
	defer os.RemoveAll(tmpBzrBranch)
	must(bzr.Clone(url, tmpBzrBranch))

	if !shouldExport(tmpBzrBranch) {
		return false
	}

	log.Info("Exporting data from bzr")
	defer func() {
		if e := recover(); e != nil {
			git.RemoveBranch(tmpGitBranch)
			panic(e)
		}
	}()
	exportSize, err := RunPipe(
		bzr.Export(tmpBzrBranch, tmpGitBranch, bzrMarks, tmpBzrMarks.Name()),
		git.Import(gitMarks, tmpGitMarks.Name()))
	must(err)

	// if all revisions of the branch are already in the repo,
	// fast-export will produce empty export and we won't
	// be able to import the branch via normal means
	if exportSize == 0 {
		log.Info("Empty export. Creating git branch using marks")
		rev, err := bzr.Tip(tmpBzrBranch)
		must(err)
		b, err := loadMarks(bzrMarks)
		must(err)
		g, err := loadMarks(gitMarks)
		must(err)
		mark, ok := b.byRev[rev]
		if !ok {
			log.Panicf("Can't find revision %q in the bzr marks file", rev)
		}
		grev, ok := g.byMark[mark]
		if !ok {
			log.Panicf("Can't find mark %d in git marks file", mark)
		}
		must(git.NewBranch(tmpGitBranch, grev))
	}

	log.Info("Finalising import")
	finalizer((exportSize != 0), tmpGitMarks.Name(), tmpBzrMarks.Name(), tmpGitBranch, tmpBzrBranch)
	return true
}

func importUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge import [-h] [-g <branch>] <url> <bzr branch>")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
import will clone bzr branch from <url> and save it as <bzr branch> in
the internal bzr repo. Then it will import the branch into git as <branch>.
If <branch> isn't specified, it is assumed to be the same as <bzr branch>
`)
}
