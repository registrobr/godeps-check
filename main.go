package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

var (
	godepsJSON, temporaryDir, git string
)

func init() {
	flag.StringVar(&godepsJSON, "godeps", os.Getenv("PWD")+"/Godeps/Godeps.json", "path to Godeps.json")
	flag.StringVar(&temporaryDir, "temp", "", "temporary path for cloning the repositories")
	flag.Parse()
}

func main() {
	f, err := ioutil.ReadFile(godepsJSON)

	if err != nil {
		fmt.Fprintf(os.Stderr, "opening %s: %s", godepsJSON, err)
		os.Exit(1)
	}

	var godeps godeps
	if err := json.Unmarshal(f, &godeps); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshalling %s: %s", godepsJSON, err)
		os.Exit(1)
	}

	if len(temporaryDir) == 0 {
		var err error
		temporaryDir, err = ioutil.TempDir("/tmp", "godeps-check")

		if err != nil {
			fmt.Fprintf(os.Stderr, "creating temporary dir: %s", err)
			os.Exit(1)
		}

		if err := os.Chmod(temporaryDir, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "chmod %s: %s", temporaryDir, err)
			os.Exit(1)
		}

		defer os.RemoveAll(temporaryDir)
	}

	git, err = exec.LookPath("git")

	if err != nil {
		fmt.Fprintf(os.Stderr, "looking up git path: %s", err)
		os.Exit(1)
	}

	godeps.process(temporaryDir, git)

	for _, dep := range godeps.Deps {
		if len(dep.commits) > 1 {
			fmt.Println(dep.ImportPath)

			for _, commit := range dep.commits {
				fmt.Println("    " + commit)
			}

			fmt.Println("")
		}
	}
}
