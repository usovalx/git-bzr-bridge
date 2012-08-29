package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func branchesCmd(args []string) {
	// command-line flags
	fs := flag.NewFlagSet("branches", flag.ExitOnError)
	help := fs.Bool("h", false, "show usage message")
	verbose := fs.Bool("v", false, "more detailed output")
	fs.Usage = func() { branchesUsage(fs) }
	fs.Parse(args)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if fs.NArg() != 0 {
		fs.Usage()
		os.Exit(2)
	}

	config, err := loadBranchConfig()
	must(err)

	branches := config.branches
	sort.Sort(byGitName{branches})

	if !*verbose {
		for _, v := range branches {
			fmt.Println(v.Git)
		}
	} else {
		for i, v := range branches {
			if i != 0 {
				fmt.Println("")
			}
			fmt.Printf("Git: %s\nUrl: %s\nBzr: %s\n", v.Git, v.Url, v.Bzr)
		}
	}
}

func branchesUsage(fs *flag.FlagSet) {
	fmt.Println("usage: git-bzr-bridge branches [-h] [-v]")
	fmt.Println("\nflags:")
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()

	fmt.Print(`
branches will list all branches known to git-bzr-bridge. Default output
format will list git branch references one per line. If -v is specified
it will also show Bzr urls and names of the (hidden) bzr branches.
`)
}

type byGitName struct{ branchList }

func (x byGitName) Less(i, j int) bool { return x.branchList[i].Git < x.branchList[j].Git }
func (x byGitName) Len() int           { return len(x.branchList) }
func (x byGitName) Swap(i, j int)      { x.branchList[i], x.branchList[j] = x.branchList[j], x.branchList[i] }
