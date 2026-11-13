# hookdoctor

**Your Claude Code hook never fired. hookdoctor tells you why in one command.**

Hooks fail silently: a command that isn't on the app's PATH, a matcher regex that doesn't compile, a settings file with a JSON typo, a hook that exits 3 when the contract wants 0 or 2. Claude Code says nothing; your automation just doesn't happen.

hookdoctor loads every hook from your user, project and local settings, validates each one, dry-runs it with a realistic payload, and names the cause:

```text
$ hookdoctor check
[error] project/PreToolUse: matcher "[bad" is not a valid regex
  fix: fix the matcher; an uncompilable matcher never matches
[error] project/SessionStart: command "/opt/hooks/notify.sh" does not exist
  fix: point the hook at the real path; remember GUI apps do not see your terminal PATH
hookdoctor: 1 hooks healthy, 2 errors, 0 warnings
```

## What it checks

**Static** - settings JSON parses, event names are real, matcher regexes compile, the command exists and is executable, hook type is supported.

**Dry run** (unless `-no-exec`) - each hook gets a synthetic payload matching its event's documented schema, then hookdoctor validates the contract: exit code semantics (0 ok, 2 blocks with stderr to the agent, anything else is a crash), stdout JSON validity, and timeouts.

Every finding names a cause and the concrete next step.

## Use it

```sh
go install github.com/ofershap/hookdoctor@v0.1.0
hookdoctor check
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-root` | `.` | Project root (project + local settings layers) |
| `-event` | all | Only check one event, e.g. `-event PreToolUse` |
| `-no-exec` | off | Static checks only, never run hook commands |
| `-settings` | - | Extra settings file to check (repeatable) |
| `-format` | `text` | `text` or `json` |
| `-max-timeout` | `30` | Per-hook dry-run cap in seconds |

Exit code is 1 when any error-level finding exists, so it gates CI over a dotfiles repo.

## What it doesn't do

- It never changes your settings or your hooks. Diagnosis only.
- Dry runs use synthetic probe payloads, clearly documented; `-no-exec` opts out of execution entirely.
- It is not a hook framework or generator - it doesn't write hooks for you.

## License

MIT
