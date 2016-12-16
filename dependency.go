package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

type dependency struct {
	ImportPath string
	Revision   string `json:"rev"`
	projectDir string
	commits    []string
}

func (d *dependency) process(git, localProvider string) {
	if err := os.MkdirAll(d.projectDir, 0777); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if err := d.clone(git, localProvider); err != nil {
		return
	}

	d.diff(git)
}

func (d *dependency) clone(git, localProvider string) error {
	url := "https://" + d.ImportPath

	if !strings.Contains(strings.Split(d.ImportPath, "/")[0], ".") {
		if localProvider == "" {
			fmt.Fprintf(os.Stderr, "skipping %s: not go gettable (no local provider)\n", d.ImportPath)
			return nil
		}

		url = localProvider + d.ImportPath
	}

	cmd := exec.Command(git, "clone", url, d.projectDir)
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

func (d *dependency) diff(git string) {
	cmd := exec.Command(git, "-C", d.projectDir, "log", "--pretty=format:%cd %h %s", "--date=format:%d/%m/%Y", d.Revision+"..master")
	output, err := cmd.CombinedOutput()
	lines := strings.Split(string(output), "\n")

	if err != nil {
		d.commits = append(lines, err.Error())
		return
	}

	if len(lines) > 10 {
		size := len(lines) - 10
		lines = append(lines[0:10], fmt.Sprintf("...                more %d commits", size))
	}

	d.commits = lines
}

func (d *dependency) Prepare(temporaryDir string) {
	parts := strings.Split(d.ImportPath, "/")

	if len(parts) > 3 {
		d.ImportPath = strings.Join(parts[0:3], "/")
	}

	d.projectDir = path.Join(temporaryDir, d.ImportPath)
}
