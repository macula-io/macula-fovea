// fovea: CyberSec-as-Code CLI for the macula-fovea spec (v0.2 to v0.6).
// Single static binary; every command runs against an assessment directory.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/macula-io/macula-fovea/cli/internal/core"
)

const usage = `fovea: CyberSec-as-Code CLI (spec v0.6)

usage: fovea <command> [dir]

commands:
  init    <dir>   create missing cell skeletons from the header
  lint    <dir>   validate header + cells, enforce anti-theater rules
  score   <dir>   lint, then compute the scorecard metrics
  render  <dir>   print the scorecard as a markdown grid
  issues  <dir>   sync roadmap/overdue cells to GitHub issues
  verify  <obs>   verify a signed claim observation offline (fovea verify --help)
  --version       the tool version and the spec versions it reads

dir defaults to the current directory. flags: --json on score and render;
--format md|html|json (or --html) on render; --github on lint;
--help-issues for the issues command. Results go to stdout, diagnostics
to stderr.`

const issuesUsage = `fovea issues <dir> [flags]

syncs roadmap/overdue cells to GitHub issues, idempotently, via a body
marker (<!-- fovea-cell: <id> -->). Issues without the marker are never
touched.

flags:
  --dry-run   compute decisions, perform no writes
  --check     fail if a roadmap cell has a closed linked issue (the trap)
  --repo o/r  target repo (default: detect from git remote origin)
  --token t   GitHub token (default: $GITHUB_TOKEN)`

func parseIssuesFlags(args []string) core.IssuesOpts {
	o := core.IssuesOpts{Dir: "."}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--dry-run":
			o.DryRun = true
		case a == "--check":
			o.Check = true
		case strings.HasPrefix(a, "--repo="):
			o.Repo = strings.TrimPrefix(a, "--repo=")
		case a == "--repo" && i+1 < len(args):
			o.Repo = args[i+1]
			i++
		case strings.HasPrefix(a, "--token="):
			o.Token = strings.TrimPrefix(a, "--token=")
		case a == "--token" && i+1 < len(args):
			o.Token = args[i+1]
			i++
		default:
			o.Dir = a
		}
	}
	return o
}

// version is the tool's version, the one place it is written. Release
// builds bake it in: go build -ldflags "-X main.version=0.1.0". It is not
// the spec version; the spec versions this build reads come from core.
var version = "dev"

// toolVersion is version, or for a build without it baked in, the module
// version `go install ...@vX.Y.Z` records; "dev" for a source build.
func toolVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && version == "dev" && strings.HasPrefix(info.Main.Version, "v") {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return version
}

// knownFlag lists every flag of init, lint, score and render; takes says
// which command accepts which. A known flag on the wrong command is refused
// rather than ignored: `lint --json` must not look like it produced JSON.
var knownFlag = map[string]bool{
	"--json": true, "--github": true, "--html": true,
	"--format=md": true, "--format=html": true, "--format=json": true,
}

var takes = map[string]map[string]bool{
	"init":   {},
	"lint":   {"--github": true},
	"score":  {"--json": true},
	"render": {"--json": true, "--html": true, "--format=md": true, "--format=html": true, "--format=json": true},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole CLI minus the process: args exclude the program name.
func run(argv []string, stdout, stderr io.Writer) int {
	if code, handled := earlyCommand(argv, stdout, stderr); handled {
		return code
	}
	opt, code, ok := parseArgs(argv[0], argv[1:], stderr)
	if !ok {
		return code
	}
	switch argv[0] {
	case "init":
		return core.Init(opt.dir, stdout, stderr)
	case "lint":
		return core.Lint(opt.dir, opt.githubOut, stdout, stderr)
	case "score":
		return core.Score(opt.dir, opt.jsonOut, stdout, stderr)
	case "render":
		return core.Render(opt.dir, opt.jsonOut, opt.htmlOut, stdout, stderr)
	case "issues":
		return core.RunIssues(parseIssuesFlags(argv[1:]), stdout, stderr)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}

// earlyCommand handles the argv[0] forms that never reach flag parsing: help,
// version and verify. handled is false when argv[0] is a normal command.
func earlyCommand(argv []string, stdout, stderr io.Writer) (int, bool) {
	switch {
	case len(argv) < 1:
		fmt.Fprintln(stderr, usage)
		return 2, true
	case argv[0] == "-h" || argv[0] == "--help":
		// Help that was asked for is output, not an error.
		fmt.Fprintln(stdout, usage)
		return 0, true
	case argv[0] == "--version":
		fmt.Fprintf(stdout, "fovea %s (reads spec %s)\n", toolVersion(), strings.Join(core.SpecVersions(), ", "))
		return 0, true
	case argv[0] == "--help-issues":
		fmt.Fprintln(stdout, issuesUsage)
		return 0, true
	case argv[0] == "verify":
		core.ToolVersion = toolVersion()
		return core.RunVerify(argv[1:], stdout, stderr), true
	}
	return 0, false
}

// options are the flags init, lint, score and render share, and the directory
// they act on.
type options struct {
	dir       string
	jsonOut   bool
	githubOut bool
	htmlOut   bool
}

// parseArgs reads those flags. A flag the command does not take is refused
// rather than ignored, so `lint --json` cannot look like it produced JSON.
func parseArgs(cmd string, args []string, stderr io.Writer) (options, int, bool) {
	opt := options{dir: "."}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--format" && i+1 < len(args) {
			i++
			a = "--format=" + args[i]
		}
		if strings.HasPrefix(a, "--") && cmd != "issues" && knownFlag[a] && !takes[cmd][a] {
			fmt.Fprintf(stderr, "fovea: %s does not take %s\n\n%s\n", cmd, a, usage)
			return opt, 2, false
		}
		switch a {
		case "--json", "--format=json":
			opt.jsonOut = true
		case "--github":
			opt.githubOut = true
		case "--html", "--format=html":
			opt.htmlOut = true
		case "--format=md":
			// markdown is render's default output
		default:
			if strings.HasPrefix(a, "--") && cmd != "issues" {
				fmt.Fprintf(stderr, "fovea: unknown flag %s\n\n%s\n", a, usage)
				return opt, 2, false
			}
			opt.dir = a
		}
	}
	return opt, 0, true
}
