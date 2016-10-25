package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

type dependency struct {
	ImportPath string
	Rev        string // VCS-specific commit ID.
}

var (
	godepsJSON = flag.String("godeps", os.Getenv("PWD")+"/Godeps/Godeps.json", "path to Godeps.json")
	godeps     struct {
		ImportPath string
		Deps       []dependency
	}
	temporaryDir, git string
)

func init() {
	flag.StringVar(&temporaryDir, "temp", "", "temporary path for cloning the repositories")
	flag.Parse()
}

func main() {
	f, err := ioutil.ReadFile(*godepsJSON)

	if err != nil {
		fmt.Fprintf(os.Stderr, "opening %s: %s", *godepsJSON, err)
		os.Exit(1)
	}

	if err := json.Unmarshal(f, &godeps); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshalling %s: %s", *godepsJSON, err)
		os.Exit(1)
	}

	if len(temporaryDir) == 0 {
		createTemporaryDir()
		defer os.RemoveAll(temporaryDir)
	}

	lookGITPath()
	results := processDependencies(godeps.Deps)

	for _, dep := range godeps.Deps {
		if commits := results[dep.ImportPath]; len(commits) > 1 {
			fmt.Println(dep.ImportPath)

			for _, commit := range commits {
				fmt.Println("  " + commit)
			}
		}
	}
}

func createTemporaryDir() {
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
}

func lookGITPath() {
	var err error
	git, err = exec.LookPath("git")

	if err != nil {
		fmt.Fprintf(os.Stderr, "looking up git path: %s", err)
		os.Exit(1)
	}
}

func processDependencies(deps []dependency) map[string][]string {
	type result struct {
		importPath string
		commits    []string
	}

	results := make(map[string][]string)
	ch := make(chan result, 0)

	for _, d := range deps {
		go func(d dependency, ch chan result) {
			ch <- result{d.ImportPath, processDependency(d)}
		}(d, ch)
	}

	done := 0

alldone:
	for {
		select {
		case r := <-ch:
			results[r.importPath] = r.commits
			done++

			if done == len(deps) {
				break alldone
			}
		}
	}

	return results
}

func processDependency(d dependency) []string {
	importPath := d.ImportPath
	parts := strings.Split(importPath, "/")

	if len(parts) > 3 {
		importPath = strings.Join(parts[0:3], "/")
	}

	if !strings.Contains(parts[0], ".") {
		fmt.Fprintf(os.Stderr, "skipping %s: not go gettable\n", d.ImportPath)
		return nil
	}

	projectDir := path.Join(temporaryDir, importPath)

	if err := os.MkdirAll(projectDir, 0777); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}

	if err := clone("https://"+importPath, projectDir); err != nil {
		return nil
	}

	return diff(projectDir, d.Rev)
}

func clone(url, projectDir string) error {
	cmd := exec.Command(git, "clone", url, projectDir)
	output, err := cmd.CombinedOutput()
	content := string(output)

	fmt.Fprint(os.Stderr, strings.Join(cmd.Args, " "))

	if err != nil {
		fmt.Fprintf(os.Stderr, ": %s\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "")
	}

	fmt.Fprintln(os.Stderr, "  "+strings.Replace(content, "\n", "\n  ", -1))

	return err
}

func diff(path, revision string) []string {
	cmd := exec.Command(git, "-C", path, "log", "--pretty=%h %s", revision+"..master")
	output, err := cmd.CombinedOutput()
	lines := strings.Split(string(output), "\n")

	if err != nil {
		return append(lines, err.Error()+"\n")
	}

	if len(lines) > 10 {
		size := len(lines) - 10
		lines = append(lines[0:9], fmt.Sprintf("[%d commits not shown]\n", size))
	}

	return lines
}
