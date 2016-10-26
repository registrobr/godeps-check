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
	provider   string
	commits    []string
}

func (d *dependency) processDependency(temporaryDir, git string) {
	projectDir := path.Join(temporaryDir, d.ImportPath)

	if err := os.MkdirAll(projectDir, 0777); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if err := d.clone(projectDir); err != nil {
		return
	}

	d.commits = d.diff(projectDir)
}

func (d *dependency) clone(projectDir string) error {
	url := "https://" + d.ImportPath
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

func (d *dependency) diff(path string) []string {
	cmd := exec.Command(git, "-C", path, "log", "--pretty=format:%cd %h %s", "--date=format:%d/%m/%Y", d.Revision+"..master")
	output, err := cmd.CombinedOutput()
	lines := strings.Split(string(output), "\n")

	if err != nil {
		return append(lines, err.Error())
	}

	if len(lines) > 10 {
		size := len(lines) - 10
		lines = append(lines[0:10], fmt.Sprintf("...                more %d commits", size))
	}

	return lines
}

func (d *dependency) Normalize() {
	parts := strings.Split(d.ImportPath, "/")

	if len(parts) > 3 {
		d.ImportPath = strings.Join(parts[0:3], "/")
	}

	d.provider = parts[0]
}
