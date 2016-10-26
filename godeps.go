package main

import (
	"fmt"
	"os"
	"strings"
)

type godeps struct {
	Deps []*dependency
}

func (g *godeps) processDependencies(temporaryDir, git string) {
	ch := make(chan bool, 0)
	names := make(map[string]bool)
	size := 0
	done := 0

	for _, dep := range g.Deps {
		dep.Normalize()

		if _, ok := names[dep.ImportPath]; ok {
			continue
		}

		if !strings.Contains(dep.provider, ".") {
			fmt.Fprintf(os.Stderr, "skipping %s: not go gettable\n", dep.ImportPath)
			continue
		}

		size++
		names[dep.ImportPath] = true

		go func(dep *dependency, ch chan bool) {
			dep.processDependency(temporaryDir, git)
			ch <- true
		}(dep, ch)
	}

alldone:
	for {
		select {
		case <-ch:
			done++

			if done == size {
				break alldone
			}
		}
	}
}
