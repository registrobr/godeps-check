package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	godepsJSON = flag.String("godeps", os.Getenv("PWD")+"/Godeps/Godeps.json", "path go Godeps.json")
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
	processDependencies(godeps.Deps)
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

func processDependencies(deps []dependency) {
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

		if err := run(os.Stderr, nil, git, "clone", url, projectDir); err != nil {
			continue
		}

		if err := os.Chdir(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "chdir %s: %s", temporaryDir, err)
			continue
		}

		run(os.Stdout, cutLog, git, "log", "--pretty=oneline", fmt.Sprintf("%s..master", d.Rev))
	}
}

func cutLog(content string) string {
	parts := strings.Split(content, "\n")

	if len(parts) > 10 {
		size := len(parts) - 10

		parts = parts[0:9]
		parts = append(parts, fmt.Sprintf("[%d more log entries]\n", size))
	}

	return strings.Join(parts, "\n")
}

func run(out io.Writer, f func(string) string, executable string, arguments ...string) error {
	cmd := exec.Command(executable, arguments...)
	output, err := cmd.CombinedOutput()
	content := string(output)

	wd, _ := os.Getwd()
	fmt.Fprintln(out, strings.Replace(wd, temporaryDir+"/", "", -1))
	fmt.Fprint(out, executable, " ", strings.Join(arguments, " "))

	if err != nil {
		fmt.Fprintf(out, ": %s\n", err)
	} else {
		fmt.Fprintln(out, "")
	}

	if f != nil {
		content = f(content)
	}

	fmt.Fprintln(out, "  "+strings.Replace(content, "\n", "\n  ", -1))

	return err
}
