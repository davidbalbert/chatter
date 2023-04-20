package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

type pager struct {
	strings.Builder
	w writeFder
}

func (p *pager) print() {
	s := p.String()
	nlines := strings.Count(s, "\n")

	_, height, err := term.GetSize(int(p.w.Fd()))
	if err != nil {
		fmt.Fprint(p.w, s)
		return
	}

	if nlines < height {
		fmt.Fprint(p.w, s)
		return
	}

	_, err = exec.LookPath("less")
	if err != nil {
		fmt.Fprint(p.w, s)
		return
	}

	cmd := exec.Command("less")
	cmd.Stdin = strings.NewReader(s)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Fprint(p.w, s)
		return
	}
}
