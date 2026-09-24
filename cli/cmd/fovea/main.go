// fovea — CyberSec-as-Code CLI for the macula-fovea spec (v0.2).
// Single static binary; every command runs against an assessment directory.
package main

import (
	"fmt"
	"os"

	"github.com/macula-io/macula-fovea/cli/internal/core"
)

const usage = `fovea — CyberSec-as-Code CLI (spec v0.2)

usage: fovea <command> [dir]

commands:
  init    <dir>   create missing cell skeletons from the header
  lint    <dir>   validate header + cells, enforce anti-theater rules
  score   <dir>   lint, then compute the scorecard metrics
  render  <dir>   print the scorecard as a markdown grid

dir defaults to the current directory. flags: --json on score/render.`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	dir := "."
	jsonOut := false
	for _, a := range os.Args[2:] {
		switch a {
		case "--json":
			jsonOut = true
		default:
			dir = a
		}
	}
	switch os.Args[1] {
	case "init":
		os.Exit(core.Init(dir))
	case "lint":
		os.Exit(core.Lint(dir))
	case "score":
		os.Exit(core.Score(dir, jsonOut))
	case "render":
		os.Exit(core.Render(dir, jsonOut))
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}
