// Package hooks validates and dry-runs Claude Code hooks.
package hooks

// settings mirrors the hooks section of a Claude Code settings.json.
type settings struct {
	Hooks map[string][]matcherGroup `json:"hooks"`
}

type matcherGroup struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// Finding is one diagnosed problem.
type Finding struct {
	Level   string // "error", "warning", "info"
	Layer   string // which settings file the hook came from
	Event   string
	Command string
	Cause   string // short machine-stable cause id
	Message string
	Fix     string // the concrete next step
}

// HookReport is the per-hook dry-run outcome.
type HookReport struct {
	Layer    string `json:"layer"`
	Event    string `json:"event"`
	Matcher  string `json:"matcher,omitempty"`
	Command  string `json:"command"`
	Ran      bool   `json:"ran"`
	ExitCode int    `json:"exit_code,omitempty"`
	StdoutOK bool   `json:"stdout_contract_ok,omitempty"`
	Skipped  string `json:"skipped,omitempty"`
}

// Result aggregates everything Check found.
type Result struct {
	Layers   []string     `json:"layers"`
	Findings []Finding    `json:"findings"`
	Reports  []HookReport `json:"reports"`
}

// events with their documented payload shapes.
var events = []string{
	"PreToolUse", "PostToolUse", "Notification", "UserPromptSubmit",
	"SessionStart", "SessionEnd", "Stop", "SubagentStop", "PreCompact",
}
