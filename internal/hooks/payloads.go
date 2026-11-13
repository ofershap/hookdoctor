package hooks

import "fmt"

// payload returns a synthetic stdin payload matching the documented schema
// for the event.
func payload(event, cwd string) string {
	base := fmt.Sprintf(`"session_id":"00000000-0000-0000-0000-000000000000","transcript_path":"/tmp/hookdoctor-transcript.jsonl","cwd":%q,"hook_event_name":%q`, cwd, event)
	switch event {
	case "PreToolUse":
		return `{` + base + `,"tool_name":"Write","tool_input":{"file_path":"/tmp/hookdoctor-probe.txt","content":"probe"}}`
	case "PostToolUse":
		return `{` + base + `,"tool_name":"Write","tool_input":{"file_path":"/tmp/hookdoctor-probe.txt","content":"probe"},"tool_response":{"filePath":"/tmp/hookdoctor-probe.txt","success":true}}`
	case "Notification":
		return `{` + base + `,"message":"hookdoctor probe"}`
	case "UserPromptSubmit":
		return `{` + base + `,"prompt":"hookdoctor probe"}`
	case "SessionStart":
		return `{` + base + `,"source":"startup"}`
	case "SessionEnd":
		return `{` + base + `,"reason":"other"}`
	case "Stop":
		return `{` + base + `,"stop_hook_active":false}`
	case "SubagentStop":
		return `{` + base + `,"stop_hook_active":false,"subagent_type":"probe"}`
	case "PreCompact":
		return `{` + base + `,"trigger":"manual","custom_instructions":""}`
	}
	return `{` + base + `}`
}
