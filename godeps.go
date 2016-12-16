package main

type godeps struct {
	Deps []*dependency
}

func (g *godeps) process(temporaryDir, git, localProvider string) {
	ch := make(chan bool, 0)
	names := make(map[string]bool)
	size := 0
	done := 0

	for _, dep := range g.Deps {
		dep.Prepare(temporaryDir)

		if _, ok := names[dep.ImportPath]; ok {
			continue
		}

		size++
		names[dep.ImportPath] = true

		go func(dep *dependency, ch chan bool) {
			dep.process(git, localProvider)
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
