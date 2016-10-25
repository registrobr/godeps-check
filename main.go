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

	if err := os.Chdir(temporaryDir); err != nil {
		fmt.Fprintf(os.Stderr, "chdir %s: %s", temporaryDir, err)
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
	results := make(map[string][]string)

	for _, d := range deps {
		importPath := d.ImportPath
		parts := strings.Split(importPath, "/")

		if len(parts) > 3 {
			importPath = strings.Join(parts[0:3], "/")
		}

		if !strings.Contains(parts[0], ".") {
			fmt.Fprintf(os.Stderr, "skipping %s: not go gettable\n", d.ImportPath)
			continue
		}

		projectDir := path.Join(temporaryDir, importPath)

		if err := os.MkdirAll(projectDir, 0777); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		// defer os.RemoveAll(projectDir)

		url := "https://" + importPath

		if err := clone(url, projectDir); err != nil {
			continue
		}

		if err := os.Chdir(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "chdir %s: %s", temporaryDir, err)
			continue
		}

		results[d.ImportPath] = diff(d.Rev)
	}

	return results
}

func clone(url, projectDir string) error {
	cmd := exec.Command(git, "clone", url, projectDir)
	output, err := cmd.CombinedOutput()
	content := string(output)

	wd, _ := os.Getwd()
	fmt.Fprintln(os.Stderr, strings.Replace(wd, temporaryDir+"/", "", -1))
	fmt.Fprint(os.Stderr, strings.Join(cmd.Args, " "))

	if err != nil {
		fmt.Fprintf(os.Stderr, ": %s\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "")
	}

	fmt.Fprintln(os.Stderr, "  "+strings.Replace(content, "\n", "\n  ", -1))

	return err
}

func diff(revision string) []string {
	cmd := exec.Command(git, "log", "--pretty=oneline", fmt.Sprintf("%s..master", revision))
	output, err := cmd.CombinedOutput()

	if err != nil {
		return []string{err.Error()}
	}

	parts := strings.Split(string(output), "\n")

	if len(parts) > 10 {
		size := len(parts) - 10

		parts = parts[0:9]
		parts = append(parts, fmt.Sprintf("[%d commits not shown]\n", size))
	}

	return parts
}
