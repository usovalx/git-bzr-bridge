package main

import (
	"gitbridge/bzr"
	"gitbridge/git"

	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path" // FIXME: should use filepath instead
)

func importCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	b := fs.String("b", "", "git branch name")
	fs.SetOutput(os.Stdout)
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

	url := fs.Arg(0)
	bzrBranch := fs.Arg(1)
	gitBranch := *b
	if gitBranch == "" {
		gitBranch = bzrBranch
	}
	bzrBranch = path.Join(bzrRepo, bzrBranch)

	// load branch config and check that resulting branch names don't clash
	if c, err := loadBranchConfig(); err != nil {
		log.Error(err)
		os.Exit(1)
	} else if c.byBzrName[bzrBranch] != nil || c.byGitName[gitBranch] != nil {
		log.Error("Requested branch names clash with existing ones")
		os.Exit(1)
	}

	// FIXME: locks

	log.Info("Cloning bzr branch")
	// FIXME: clone into temp location first
	must(os.MkdirAll(path.Dir(bzrBranch), 0777))
	must(bzr.Clone(url, bzrBranch))

	log.Info("Exporting data from bzr")
	tmpData, err1 := ioutil.TempFile(tmpDir, "bzr_export")
	tmpBzrMarks, err2 := ioutil.TempFile(tmpDir, "bzr_marks")
	tmpGitBranch := tempGitBranch()
	must(err1)
	must(err2)
	gzw := gzip.NewWriter(tmpData)
	must(bzr.Export(bzrBranch, tmpGitBranch, gzw, bzrMarks, tmpBzrMarks.Name()))
	gzw.Close()

	log.Info("Importing data into git")
	tmpGitMarks, err3 := ioutil.TempFile(tmpDir, "git_marks")
	must(err3)
	tmpData.Seek(0, os.SEEK_SET)
	gzr, err4 := gzip.NewReader(tmpData)
	must(err4)
	must(git.Import(gzr, gitMarks, tmpGitMarks.Name()))

	log.Info("Applying changes")
	// TODO: move bzr branch from temp to where it should be
	must(git.RenameBranch(gitBranch, tmpGitBranch))
	must(addBranchToConfig(url, bzrBranch, gitBranch))
	must(moveFile(tmpBzrMarks.Name(), bzrMarks))
	must(moveFile(tmpGitMarks.Name(), gitMarks))

	os.Exit(0)
}

func importUsage(fs *flag.FlagSet) {
	fmt.Println("usage: gitbridge import [-h] [-g <branch>] <url> <bzr branch>")
	fmt.Println("\nflags:")
	fs.PrintDefaults()

	fmt.Print(`
import will clone bzr branch from <url> and save it as <bzr branch> in
the internal bzr repo. Then it will import the branch into git as <branch>.
If <branch> isn't specified, it is assumed to be the same as <bzr branch>
`)
}
