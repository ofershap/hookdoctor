package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Options configures Check.
type Options struct {
	Root       string
	ExtraFiles []string
	Event      string
	Exec       bool // opt-in: actually run hook commands
	MaxTimeout int
}

type layeredFile struct {
	layer string
	path  string
}

// layers returns the settings files to inspect, in precedence order.
func (o Options) layers() []layeredFile {
	home, _ := os.UserHomeDir()
	out := []layeredFile{
		{layer: "user", path: filepath.Join(home, ".claude", "settings.json")},
		{layer: "project", path: filepath.Join(o.Root, ".claude", "settings.json")},
		{layer: "local", path: filepath.Join(o.Root, ".claude", "settings.local.json")},
	}
	for _, f := range o.ExtraFiles {
		out = append(out, layeredFile{layer: "extra:" + filepath.Base(f), path: f})
	}
	return out
}

// Check runs all validations and dry-runs.
func Check(opts Options) (Result, error) {
	res := Result{}
	anyFile := false
	for _, lf := range opts.layers() {
		raw, err := os.ReadFile(lf.path)
		if err != nil {
			continue // layer simply not present
		}
		anyFile = true
		res.Layers = append(res.Layers, lf.layer+" ("+lf.path+")")

		var s settings
		if err := json.Unmarshal(raw, &s); err != nil {
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: lf.layer, Cause: "invalid-json",
				Message: fmt.Sprintf("%s does not parse: %v", lf.path, err),
				Fix:     "fix the JSON; hooks in an unparsable settings file are silently dropped",
			})
			continue
		}
		if s.Hooks == nil {
			continue
		}
		for _, event := range events {
			if opts.Event != "" && !strings.EqualFold(opts.Event, event) {
				continue
			}
			groups, ok := s.Hooks[event]
			if !ok {
				continue
			}
			for _, g := range groups {
				checkMatcher(&res, lf.layer, event, g.Matcher)
				for _, h := range g.Hooks {
					res.Reports = append(res.Reports, checkHook(&res, opts, lf.layer, event, g.Matcher, h)...)
				}
			}
		}
		// unknown event names in the file
		for name := range s.Hooks {
			known := false
			for _, e := range events {
				if e == name {
					known = true
				}
			}
			if !known {
				res.Findings = append(res.Findings, Finding{
					Level: "warning", Layer: lf.layer, Event: name, Cause: "unknown-event",
					Message: fmt.Sprintf("%q is not a known hook event", name),
					Fix:     "check the event name against the hooks reference; unknown events never fire",
				})
			}
		}
	}
	if !anyFile {
		return res, fmt.Errorf("no settings files found (looked in user/project/local layers under %s)", opts.Root)
	}
	return res, nil
}

func checkMatcher(res *Result, layer, event, matcher string) {
	if matcher == "" {
		return
	}
	if event == "PreToolUse" || event == "PostToolUse" || event == "Notification" {
		if _, err := regexp.Compile(matcher); err != nil {
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: layer, Event: event, Cause: "bad-matcher",
				Message: fmt.Sprintf("matcher %q is not a valid regex: %v", matcher, err),
				Fix:     "fix the matcher; an uncompilable matcher never matches",
			})
		}
	}
}

// checkHook validates one hook entry and dry-runs it unless disabled.
func checkHook(res *Result, opts Options, layer, event, matcher string, h hookEntry) []HookReport {
	rep := HookReport{Layer: layer, Event: event, Matcher: matcher, Command: h.Command}

	if h.Type != "" && h.Type != "command" {
		res.Findings = append(res.Findings, Finding{
			Level: "error", Layer: layer, Event: event, Command: h.Command, Cause: "bad-type",
			Message: fmt.Sprintf("hook type %q is not supported (only \"command\")", h.Type),
			Fix:     "set \"type\": \"command\"",
		})
		rep.Skipped = "bad-type"
		return []HookReport{rep}
	}
	if strings.TrimSpace(h.Command) == "" {
		res.Findings = append(res.Findings, Finding{
			Level: "error", Layer: layer, Event: event, Cause: "empty-command",
			Message: "hook has an empty command",
			Fix:     "remove the entry or give it a command",
		})
		rep.Skipped = "empty-command"
		return []HookReport{rep}
	}

	// command resolution: first word as file or on PATH
	first := strings.Fields(h.Command)[0]
	if strings.Contains(first, "/") {
		st, err := os.Stat(first)
		switch {
		case err != nil:
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: layer, Event: event, Command: h.Command, Cause: "command-not-found",
				Message: fmt.Sprintf("command %q does not exist", first),
				Fix:     "point the hook at the real path; remember GUI apps do not see your terminal PATH",
			})
			rep.Skipped = "command-not-found"
			return []HookReport{rep}
		case st.IsDir():
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: layer, Event: event, Command: h.Command, Cause: "command-is-dir",
				Message: fmt.Sprintf("command %q is a directory", first),
				Fix:     "point the hook at an executable file",
			})
			rep.Skipped = "command-is-dir"
			return []HookReport{rep}
		case st.Mode()&0o111 == 0:
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: layer, Event: event, Command: h.Command, Cause: "not-executable",
				Message: fmt.Sprintf("command %q is not executable", first),
				Fix:     "chmod +x it, or invoke its interpreter explicitly (bash script.sh)",
			})
			rep.Skipped = "not-executable"
			return []HookReport{rep}
		}
	} else if _, err := exec.LookPath(first); err != nil {
		res.Findings = append(res.Findings, Finding{
			Level: "error", Layer: layer, Event: event, Command: h.Command, Cause: "command-not-found",
			Message: fmt.Sprintf("command %q is not on PATH", first),
			Fix:     "use an absolute path; apps launched outside your shell often have a minimal PATH",
		})
		rep.Skipped = "command-not-found"
		return []HookReport{rep}
	}

	if !opts.Exec {
		rep.Skipped = "static-only (pass -exec to dry-run)"
		return []HookReport{rep}
	}
	return runHook(res, opts, rep)
}

// runHook dry-runs one hook with a synthetic payload and checks the contract.
func runHook(res *Result, opts Options, rep HookReport) []HookReport {
	max := time.Duration(opts.MaxTimeout) * time.Second
	if max <= 0 {
		max = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), max)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", rep.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// kill the whole process group so children of the hook cannot survive
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.Stdin = strings.NewReader(payload(rep.Event, "."))
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	rep.Ran = true

	if ctx.Err() == context.DeadlineExceeded {
		res.Findings = append(res.Findings, Finding{
			Level: "error", Layer: rep.Layer, Event: rep.Event, Command: rep.Command, Cause: "timeout",
			Message: fmt.Sprintf("hook did not finish within %s", max),
			Fix:     "hooks block the agent loop; keep them fast or set an explicit shorter timeout",
		})
		return []HookReport{rep}
	}

	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: rep.Layer, Event: rep.Event, Command: rep.Command, Cause: "run-failed",
				Message: fmt.Sprintf("hook could not start: %v", err),
				Fix:     "verify the command runs standalone: " + rep.Command,
			})
			return []HookReport{rep}
		}
	}
	rep.ExitCode = code

	out := strings.TrimSpace(stdout.String())
	rep.StdoutOK = true
	if code == 0 && out != "" && strings.HasPrefix(out, "{") {
		var v map[string]any
		if json.Unmarshal([]byte(out), &v) != nil {
			rep.StdoutOK = false
			res.Findings = append(res.Findings, Finding{
				Level: "error", Layer: rep.Layer, Event: rep.Event, Command: rep.Command, Cause: "invalid-json-output",
				Message: "stdout starts with { but is not valid JSON",
				Fix:     "emit a complete JSON object, or plain text only",
			})
		}
	}
	switch code {
	case 0:
		// healthy
	case 2:
		res.Findings = append(res.Findings, Finding{
			Level: "info", Layer: rep.Layer, Event: rep.Event, Command: rep.Command, Cause: "blocking-exit",
			Message: "hook exits 2 on the probe payload: it would block this action and show stderr to the agent",
			Fix:     "if that is intended for the probe, ignore; otherwise check the hook's conditions",
		})
	default:
		res.Findings = append(res.Findings, Finding{
			Level: "error", Layer: rep.Layer, Event: rep.Event, Command: rep.Command, Cause: "crash",
			Message: fmt.Sprintf("hook exited %d: %s", code, firstLine(stderr.String())),
			Fix:     "a non-zero, non-2 exit is a hook error; the agent sees it as a hook failure",
		})
	}
	return []HookReport{rep}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
