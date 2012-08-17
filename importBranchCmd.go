package main

import (
	"gitbridge/bzr"
	"gitbridge/git"

	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type countWriter uint64

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

	url := fs.Arg(0)
	bzrBranch := filepath.FromSlash(path.Join(bzrRepo, fs.Arg(1)))
	gitBranch := *b
	if gitBranch == "" {
		gitBranch = fs.Arg(1)
	}

	tmpGitBranch := tempBranchName()
	tmpBzrBranch := filepath.FromSlash(path.Join(bzrRepo, tmpGitBranch))

	// load branch config and check that resulting branch names don't clash
	c, err := loadBranchConfig()
	must(err)
	if c.byBzrName[bzrBranch] != nil || c.byGitName[gitBranch] != nil {
		log.Error("Requested branch names clash with existing ones")
		os.Exit(1)
	}

	// FIXME: use locks

	// Create all temporary files we will use later
	tmpData, err := ioutil.TempFile(tmpDir, "bzr_export")
	must(err)
	defer os.Remove(tmpData.Name())
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

	log.Info("Exporting data from bzr")
	gzw := gzip.NewWriter(tmpData)
	cw := new(countWriter)
	must(bzr.Export(tmpBzrBranch, tmpGitBranch,
		io.MultiWriter(gzw, cw), bzrMarks, tmpBzrMarks.Name()))
	gzw.Close()

	// if all revisions of the branch are already in the repo,
	// fast-export will produce empty export and we won't
	// be able to import the branch via normal means
	empty_export := (0 == *cw)

	if empty_export {
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
		defer func() {
			if e := recover(); e != nil {
				git.RemoveBranch(tmpGitBranch)
				panic(e)
			}
		}()
		must(git.NewBranch(tmpGitBranch, grev))
	} else {
		log.Info("Importing data into git")
		tmpData.Seek(0, os.SEEK_SET)
		gzr, err := gzip.NewReader(tmpData)
		must(err)
		defer func() {
			if e := recover(); e != nil {
				git.RemoveBranch(tmpGitBranch)
				panic(e)
			}
		}()
		must(git.Import(gzr, gitMarks, tmpGitMarks.Name()))
	}

	log.Info("Finalising import")
	// correct ordering of actions is extremely important here
	// while we can live with stale temporary branches and/or files
	// and easily clean them up manually later it is extremely
	// important that we keep marks files in sync.
	must(os.MkdirAll(filepath.Dir(bzrBranch), 0777))
	must(os.Rename(tmpBzrBranch, bzrBranch))
	must(git.RenameBranch(tmpGitBranch, gitBranch))
	must(addBranchToConfig(url, bzrBranch, gitBranch))
	if !empty_export {
		must(os.Rename(tmpBzrMarks.Name(), bzrMarks))
		must(os.Rename(tmpGitMarks.Name(), gitMarks))
	}
}

func importUsage(fs *flag.FlagSet) {
	fmt.Println("usage: gitbridge import [-h] [-g <branch>] <url> <bzr branch>")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
import will clone bzr branch from <url> and save it as <bzr branch> in
the internal bzr repo. Then it will import the branch into git as <branch>.
If <branch> isn't specified, it is assumed to be the same as <bzr branch>
`)
}

func (w *countWriter) Write(b []byte) (int, error) {
	n := len(b)
	*w += countWriter(n)
	return n, nil
}
