// Command hookdoctor answers "why didn't my Claude Code hook fire?": it
// validates every hook in your user, project and local settings, dry-runs
// each one with a realistic payload, and names the cause of failure.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ofershap/hookdoctor/internal/hooks"
)

var version = "dev"

func usage() {
	fmt.Fprintf(os.Stderr, `hookdoctor %s - why didn't my Claude Code hook fire?

Usage:
  hookdoctor check [-root DIR] [-settings FILE]... [-event NAME] [-exec] [-format text|json]

Checks every hook in user (~/.claude/settings.json), project
(.claude/settings.json) and local (.claude/settings.local.json) layers:
command resolution, matcher sanity. With -exec, also dry-runs each hook
with a synthetic payload and validates the output contract. -exec runs the
hook commands, which are arbitrary code: enable it only for repos you trust.

Exit codes: 0 = healthy, 1 = findings, 2 = usage error.
`, version)
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println("hookdoctor", version)
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "check" {
		usage()
		os.Exit(2)
	}

	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	root := fs.String("root", ".", "project root")
	event := fs.String("event", "", "only check this event (e.g. PreToolUse)")
	exec := fs.Bool("exec", false, "dry-run hooks with synthetic payloads (executes the hook commands; static checks only without it)")
	format := fs.String("format", "text", "output format: text or json")
	maxTimeout := fs.Int("max-timeout", 30, "cap per-hook dry-run timeout in seconds")
	var extra multiFlag
	fs.Var(&extra, "settings", "extra settings file to check (repeatable)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	res, err := hooks.Check(hooks.Options{
		Root:       *root,
		ExtraFiles: extra,
		Event:      *event,
		Exec:       *exec,
		MaxTimeout: *maxTimeout,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hookdoctor:", err)
		os.Exit(2)
	}

	switch *format {
	case "text":
		hooks.PrintText(os.Stdout, res)
	case "json":
		if err := hooks.PrintJSON(os.Stdout, res); err != nil {
			fmt.Fprintln(os.Stderr, "hookdoctor: json:", err)
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, "hookdoctor: unknown -format", *format)
		os.Exit(2)
	}

	for _, f := range res.Findings {
		if f.Level == "error" {
			os.Exit(1)
		}
	}
}

type multiFlag []string

func (m *multiFlag) String() string { return fmt.Sprint([]string(*m)) }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}
