package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, rel, content string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func check(t *testing.T, opts Options) Result {
	t.Helper()
	res, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func causes(res Result) map[string]int {
	out := map[string]int{}
	for _, f := range res.Findings {
		out[f.Cause]++
	}
	return out
}

func TestHealthyHook(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"PreToolUse":[{"matcher":"Write","hooks":[{"type":"command","command":"cat > /dev/null"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, MaxTimeout: 10})
	if len(res.Reports) != 1 || !res.Reports[0].Ran || res.Reports[0].ExitCode != 0 {
		t.Fatalf("want one healthy ran hook, got %+v (findings %+v)", res.Reports, res.Findings)
	}
	for _, f := range res.Findings {
		if f.Level == "error" {
			t.Fatalf("unexpected error finding: %+v", f)
		}
	}
}

func TestCommandNotFound(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"SessionStart":[{"matcher":"","hooks":[{"type":"command","command":"/nonexistent/hook.sh --flag"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, MaxTimeout: 10})
	if causes(res)["command-not-found"] != 1 {
		t.Fatalf("want command-not-found, got %+v", res.Findings)
	}
	if res.Reports[0].Skipped != "command-not-found" {
		t.Fatalf("hook should be skipped, got %+v", res.Reports[0])
	}
}

func TestNotExecutable(t *testing.T) {
	dir := t.TempDir()
	script := write(t, dir, "hook.sh", "#!/bin/sh\nexit 0\n", 0o644) // no +x
	write(t, dir, ".claude/settings.json", `{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"`+script+`"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, MaxTimeout: 10})
	if causes(res)["not-executable"] != 1 {
		t.Fatalf("want not-executable, got %+v", res.Findings)
	}
}

func TestCrashExit(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo boom >&2; exit 1"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, MaxTimeout: 10})
	c := causes(res)
	if c["crash"] != 1 {
		t.Fatalf("want crash, got %+v", res.Findings)
	}
	if !strings.Contains(res.Findings[0].Message, "boom") {
		t.Fatalf("crash message should carry stderr: %s", res.Findings[0].Message)
	}
}

func TestBlockingExit2IsInfo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"echo no >&2; exit 2"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, MaxTimeout: 10})
	for _, f := range res.Findings {
		if f.Level == "error" {
			t.Fatalf("exit 2 must not be an error, got %+v", f)
		}
	}
	if causes(res)["blocking-exit"] != 1 {
		t.Fatalf("want blocking-exit info, got %+v", res.Findings)
	}
}

func TestInvalidJSONOutput(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"echo '{broken'"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, MaxTimeout: 10})
	if causes(res)["invalid-json-output"] != 1 {
		t.Fatalf("want invalid-json-output, got %+v", res.Findings)
	}
}

func TestUnknownEvent(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"BeforeTool":[{"hooks":[{"type":"command","command":"true"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, NoExec: true, MaxTimeout: 10})
	if causes(res)["unknown-event"] != 1 {
		t.Fatalf("want unknown-event, got %+v", res.Findings)
	}
}

func TestBadMatcher(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{"hooks":{"PreToolUse":[{"matcher":"[unclosed","hooks":[{"type":"command","command":"true"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, NoExec: true, MaxTimeout: 10})
	if causes(res)["bad-matcher"] != 1 {
		t.Fatalf("want bad-matcher, got %+v", res.Findings)
	}
}

func TestInvalidSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".claude/settings.json", `{not json`, 0o644)

	res := check(t, Options{Root: dir, NoExec: true, MaxTimeout: 10})
	if causes(res)["invalid-json"] != 1 {
		t.Fatalf("want invalid-json, got %+v", res.Findings)
	}
}

func TestNoSettingsFiles(t *testing.T) {
	dir := t.TempDir()
	// point HOME at an empty dir so the user layer is absent too
	t.Setenv("HOME", dir)
	if _, err := Check(Options{Root: dir}); err == nil {
		t.Fatal("want error when no settings file exists anywhere")
	}
}

func TestNoExecSkipsRun(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	write(t, dir, ".claude/settings.json", `{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"touch `+marker+`"}]}]}}`, 0o644)

	res := check(t, Options{Root: dir, NoExec: true, MaxTimeout: 10})
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("--no-exec must not run the hook")
	}
	if res.Reports[0].Skipped != "no-exec" {
		t.Fatalf("want skipped=no-exec, got %+v", res.Reports[0])
	}
}

func TestPayloadShapes(t *testing.T) {
	for _, e := range events {
		p := payload(e, "/tmp")
		if !strings.Contains(p, `"hook_event_name":"`+e+`"`) {
			t.Fatalf("payload for %s missing event name: %s", e, p)
		}
		if !strings.HasPrefix(p, "{") || !strings.HasSuffix(p, "}") {
			t.Fatalf("payload for %s is not an object: %s", e, p)
		}
	}
}
