package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"krellin/internal/agents"
	"krellin/internal/capsule"
	"krellin/internal/protocol"
)

const (
	toolMaxRounds   = 8
	toolOutputLimit = 64 * 1024
	toolTimeout     = 30 * time.Second
)

type toolCall struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

type toolEnvelope struct {
	ToolCalls []toolCall      `json:"tool_calls"`
	ToolCall  *toolCall       `json:"tool_call"`
	Tool      string          `json:"tool"`
	Args      json.RawMessage `json:"args"`
	Final     string          `json:"final"`
}

type toolResult struct {
	Tool     string
	Args     string
	Output   string
	ExitCode int
	Error    string
}

func (h SessionHandler) runAgentWithTools(ctx context.Context, action protocol.Action, provider agents.Provider, content string) (string, error) {
	results := []toolResult{}
	for round := 0; round < toolMaxRounds; round++ {
		prompt := buildToolPrompt(content, results)
		resp, err := h.Session.agentsRunner.Prompt(ctx, provider, prompt)
		if err != nil {
			return "", h.emitAgentMessage(action, fmt.Sprintf("Agent error: %v", err))
		}
		toolCalls, final, ok := parseToolResponse(resp)
		if final != "" {
			return final, h.emitAgentMessage(action, final)
		}
		if !ok || len(toolCalls) == 0 {
			return resp, h.emitAgentMessage(action, resp)
		}
		for _, call := range toolCalls {
			res := h.executeTool(ctx, action, call)
			results = append(results, res)
		}
	}
	limitMsg := "Agent exceeded tool call limit."
	return limitMsg, h.emitAgentMessage(action, limitMsg)
}

func buildToolPrompt(userContent string, results []toolResult) string {
	var out strings.Builder
	out.WriteString("You are a coding agent running inside a containerized capsule. ")
	out.WriteString("You have tool access and must operate only inside the capsule. ")
	out.WriteString("When you need a tool, respond with JSON only.\n\n")
	out.WriteString("Tool call format:\n")
	out.WriteString(`{"tool_calls":[{"tool":"shell","args":{"command":"ls","cwd":"/workspace"}}]}` + "\n")
	out.WriteString("Final response format:\n")
	out.WriteString(`{"final":"<your response>"}` + "\n\n")
	out.WriteString("Tools:\n")
	out.WriteString("- shell: {command, cwd?, env?}\n")
	out.WriteString("- read_file: {path}\n")
	out.WriteString("- write_file: {path, content}\n")
	out.WriteString("- list_files: {path?, recursive?, max_depth?}\n")
	out.WriteString("- search: {pattern, path?, max_results?}\n")
	out.WriteString("- apply_patch: {patch}\n\n")
	out.WriteString("Rules:\n")
	out.WriteString("- Operate only within /workspace.\n")
	out.WriteString("- Do not use git for edits or revert.\n")
	out.WriteString("- Keep tool output concise.\n\n")
	out.WriteString("User:\n")
	out.WriteString(userContent)
	if len(results) > 0 {
		out.WriteString("\n\nTool results:\n")
		for i, res := range results {
			out.WriteString(fmt.Sprintf("%d) tool=%s args=%s exit_code=%d\n", i+1, res.Tool, res.Args, res.ExitCode))
			if res.Error != "" {
				out.WriteString("error: " + res.Error + "\n")
			}
			if res.Output != "" {
				out.WriteString("output:\n")
				out.WriteString(res.Output)
				if !strings.HasSuffix(res.Output, "\n") {
					out.WriteString("\n")
				}
			}
		}
	}
	out.WriteString("\nRespond with JSON only.\n")
	return out.String()
}

func parseToolResponse(resp string) ([]toolCall, string, bool) {
	resp = normalizeAgentResponse(resp)
	if resp == "" {
		return nil, "", false
	}
	var env toolEnvelope
	if decodeFirstJSON(resp, &env) {
		if env.Final != "" {
			return nil, env.Final, true
		}
		if len(env.ToolCalls) > 0 {
			return env.ToolCalls, "", true
		}
		if env.ToolCall != nil {
			return []toolCall{*env.ToolCall}, "", true
		}
		if env.Tool != "" {
			return []toolCall{{Tool: env.Tool, Args: env.Args}}, "", true
		}
		// Support {"shell":{...}} style.
		var single map[string]json.RawMessage
		if decodeFirstJSON(resp, &single) {
			if len(single) == 1 {
				for k, v := range single {
					return []toolCall{{Tool: k, Args: v}}, "", true
				}
			}
			if raw, ok := single["final"]; ok {
				var final string
				if err := json.Unmarshal(raw, &final); err == nil && final != "" {
					return nil, final, true
				}
			}
		}
		return nil, "", false
	}
	// Fallback: try to extract a "final" field if JSON parse failed.
	if strings.Contains(resp, "\"final\"") {
		var anyMap map[string]any
		if decodeFirstJSON(resp, &anyMap) {
			if v, ok := anyMap["final"]; ok {
				if s, ok := v.(string); ok && s != "" {
					return nil, s, true
				}
			}
		}
	}
	return nil, "", false
}

func normalizeAgentResponse(resp string) string {
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return resp
	}
	// Strip fenced code blocks like ```json ... ```
	if strings.HasPrefix(resp, "```") {
		resp = strings.TrimPrefix(resp, "```")
		resp = strings.TrimPrefix(resp, "json")
		resp = strings.TrimPrefix(resp, "JSON")
		resp = strings.TrimSpace(resp)
		if idx := strings.LastIndex(resp, "```"); idx != -1 {
			resp = resp[:idx]
		}
		resp = strings.TrimSpace(resp)
	}
	// If it looks like a bare "final":"..." pair, wrap it.
	if strings.HasPrefix(resp, "\"final\"") || strings.HasPrefix(resp, "final") {
		if !strings.HasPrefix(resp, "{") {
			resp = "{" + resp + "}"
		}
	}
	return resp
}

func decodeFirstJSON(text string, dst any) bool {
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		dec := json.NewDecoder(strings.NewReader(text[i:]))
		dec.UseNumber()
		if err := dec.Decode(dst); err == nil {
			return true
		}
	}
	return false
}

func (h SessionHandler) executeTool(ctx context.Context, action protocol.Action, call toolCall) toolResult {
	res := toolResult{Tool: call.Tool, Args: string(call.Args)}
	if h.Session == nil || h.Session.capsule == nil {
		res.Error = "capsule not configured"
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	}
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	switch call.Tool {
	case "shell":
		var args struct {
			Command string            `json:"command"`
			Cwd     string            `json:"cwd,omitempty"`
			Env     map[string]string `json:"env,omitempty"`
		}
		if err := json.Unmarshal(call.Args, &args); err != nil || strings.TrimSpace(args.Command) == "" {
			res.Error = "invalid shell args"
			return res
		}
		cwd, err := workspacePath(args.Cwd)
		if err != nil {
			cwd = "/workspace"
		}
		r, err := h.Session.capsule.Exec(ctx, h.Session.handle, args.Command, capsule.ExecOptions{Cwd: cwd, Env: args.Env})
		res.Output = truncateOutput(r.Output)
		res.ExitCode = r.ExitCode
		if err != nil {
			res.Error = err.Error()
		}
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	case "read_file":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(call.Args, &args); err != nil || args.Path == "" {
			res.Error = "invalid read_file args"
			return res
		}
		target, err := workspacePath(args.Path)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		cmd := fmt.Sprintf("cat %s", shellQuote(target))
		r, err := h.Session.capsule.Exec(ctx, h.Session.handle, cmd, capsule.ExecOptions{Cwd: "/workspace"})
		res.Output = truncateOutput(r.Output)
		res.ExitCode = r.ExitCode
		if err != nil {
			res.Error = err.Error()
		}
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	case "write_file":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(call.Args, &args); err != nil || args.Path == "" {
			res.Error = "invalid write_file args"
			return res
		}
		target, err := workspacePath(args.Path)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		dir := path.Dir(target)
		enc := base64.StdEncoding.EncodeToString([]byte(args.Content))
		cmd := fmt.Sprintf("mkdir -p %s && if command -v base64 >/dev/null 2>&1; then printf %s | base64 -d > %s; elif command -v python3 >/dev/null 2>&1; then B64=%s python3 - <<'PY' > %s\nimport os,base64,sys\nsys.stdout.write(base64.b64decode(os.environ.get('B64','')).decode('utf-8', errors='replace'))\nPY\nelif command -v python >/dev/null 2>&1; then B64=%s python - <<'PY' > %s\nimport os,base64,sys\nsys.stdout.write(base64.b64decode(os.environ.get('B64','')).decode('utf-8', errors='replace'))\nPY\nelse printf %s > %s; fi",
			shellQuote(dir),
			shellQuote(enc), shellQuote(target),
			shellQuote(enc), shellQuote(target),
			shellQuote(enc), shellQuote(target),
			shellQuote(args.Content), shellQuote(target),
		)
		r, err := h.Session.capsule.Exec(ctx, h.Session.handle, cmd, capsule.ExecOptions{Cwd: "/workspace"})
		res.Output = truncateOutput(r.Output)
		res.ExitCode = r.ExitCode
		if err != nil {
			res.Error = err.Error()
		}
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	case "list_files":
		var args struct {
			Path      string `json:"path,omitempty"`
			Recursive bool   `json:"recursive,omitempty"`
			MaxDepth  int    `json:"max_depth,omitempty"`
		}
		if err := json.Unmarshal(call.Args, &args); err != nil {
			res.Error = "invalid list_files args"
			return res
		}
		target, err := workspacePath(args.Path)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		cmd := ""
		if args.Recursive {
			depth := args.MaxDepth
			if depth <= 0 {
				depth = 4
			}
			cmd = fmt.Sprintf("find %s -maxdepth %d -print", shellQuote(target), depth)
		} else {
			cmd = fmt.Sprintf("ls -a %s", shellQuote(target))
		}
		r, err := h.Session.capsule.Exec(ctx, h.Session.handle, cmd, capsule.ExecOptions{Cwd: "/workspace"})
		res.Output = truncateOutput(r.Output)
		res.ExitCode = r.ExitCode
		if err != nil {
			res.Error = err.Error()
		}
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	case "search":
		var args struct {
			Pattern    string `json:"pattern"`
			Path       string `json:"path,omitempty"`
			MaxResults int    `json:"max_results,omitempty"`
		}
		if err := json.Unmarshal(call.Args, &args); err != nil || args.Pattern == "" {
			res.Error = "invalid search args"
			return res
		}
		target, err := workspacePath(args.Path)
		if err != nil {
			res.Error = err.Error()
			return res
		}
		limit := args.MaxResults
		if limit <= 0 {
			limit = 200
		}
		pat := shellQuote(args.Pattern)
		cmd := fmt.Sprintf("if command -v rg >/dev/null 2>&1; then rg --line-number --max-count %d %s %s; else grep -R -n -m %d %s %s; fi",
			limit, pat, shellQuote(target), limit, pat, shellQuote(target))
		r, err := h.Session.capsule.Exec(ctx, h.Session.handle, cmd, capsule.ExecOptions{Cwd: "/workspace"})
		res.Output = truncateOutput(r.Output)
		res.ExitCode = r.ExitCode
		if err != nil {
			res.Error = err.Error()
		}
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	case "apply_patch":
		var args struct {
			Patch string `json:"patch"`
		}
		if err := json.Unmarshal(call.Args, &args); err != nil || strings.TrimSpace(args.Patch) == "" {
			res.Error = "invalid apply_patch args"
			return res
		}
		delimiter := pickDelimiter(args.Patch)
		cmd := fmt.Sprintf("cd /workspace && PATCH_CONTENT=$(cat <<'%s'\n%s\n%s\n) && if command -v patch >/dev/null 2>&1; then printf \"%%s\" \"$PATCH_CONTENT\" | patch -p1; elif command -v python3 >/dev/null 2>&1; then printf \"%%s\" \"$PATCH_CONTENT\" | python3 - <<'PY'\nimport sys\npatch = sys.stdin.read().splitlines()\nfiles = []\nhunks = {}\ncurrent = None\n\ndef parse_range(r):\n    parts = r.split(',')\n    start = int(parts[0])\n    lines = int(parts[1]) if len(parts) > 1 else 1\n    return start, lines\n\nlines = patch\nidx = 0\nwhile idx < len(lines):\n    line = lines[idx]\n    if line.startswith('+++ b/'):\n        current = line[6:]\n        files.append(current)\n        if current not in hunks:\n            hunks[current] = []\n    elif line.startswith('@@'):\n        if not current:\n            raise SystemExit('hunk without file')\n        parts = line.split(' ')\n        old_range = parts[1][1:]\n        new_range = parts[2][1:]\n        old_start, old_lines = parse_range(old_range)\n        new_start, new_lines = parse_range(new_range)\n        h = {'old_start': old_start, 'old_lines': old_lines, 'new_start': new_start, 'new_lines': new_lines, 'lines': []}\n        idx += 1\n        while idx < len(lines) and not lines[idx].startswith('@@') and not lines[idx].startswith('+++ b/'):\n            h['lines'].append(lines[idx])\n            idx += 1\n        hunks[current].append(h)\n        continue\n    idx += 1\n\nif not files:\n    raise SystemExit('no files in patch')\n\nfor file in files:\n    path = file\n    with open(path, 'r', encoding='utf-8', errors='replace') as f:\n        content = f.read().splitlines()\n    offset = 0\n    for h in hunks[file]:\n        idx = h['old_start'] - 1 + offset\n        if idx < 0 or idx > len(content):\n            raise SystemExit('hunk out of range')\n        new_lines = []\n        removed = 0\n        added = 0\n        cursor = idx\n        for l in h['lines']:\n            if l == '':\n                continue\n            prefix = l[:1]\n            text = l[1:]\n            if prefix == ' ':\n                if cursor >= len(content) or content[cursor] != text:\n                    raise SystemExit('context mismatch')\n                new_lines.append(content[cursor])\n                cursor += 1\n            elif prefix == '-':\n                if cursor >= len(content) or content[cursor] != text:\n                    raise SystemExit('delete mismatch')\n                cursor += 1\n                removed += 1\n            elif prefix == '+':\n                new_lines.append(text)\n                added += 1\n            else:\n                raise SystemExit('invalid hunk line')\n        content = content[:idx] + new_lines + content[cursor:]\n        offset += added - removed\n    with open(path, 'w', encoding='utf-8') as f:\n        f.write('\\n'.join(content) + '\\n')\nPY\nelif command -v python >/dev/null 2>&1; then printf \"%%s\" \"$PATCH_CONTENT\" | python - <<'PY'\nimport sys\npatch = sys.stdin.read().splitlines()\nfiles = []\nhunks = {}\ncurrent = None\n\ndef parse_range(r):\n    parts = r.split(',')\n    start = int(parts[0])\n    lines = int(parts[1]) if len(parts) > 1 else 1\n    return start, lines\n\nlines = patch\nidx = 0\nwhile idx < len(lines):\n    line = lines[idx]\n    if line.startswith('+++ b/'):\n        current = line[6:]\n        files.append(current)\n        if current not in hunks:\n            hunks[current] = []\n    elif line.startswith('@@'):\n        if not current:\n            raise SystemExit('hunk without file')\n        parts = line.split(' ')\n        old_range = parts[1][1:]\n        new_range = parts[2][1:]\n        old_start, old_lines = parse_range(old_range)\n        new_start, new_lines = parse_range(new_range)\n        h = {'old_start': old_start, 'old_lines': old_lines, 'new_start': new_start, 'new_lines': new_lines, 'lines': []}\n        idx += 1\n        while idx < len(lines) and not lines[idx].startswith('@@') and not lines[idx].startswith('+++ b/'):\n            h['lines'].append(lines[idx])\n            idx += 1\n        hunks[current].append(h)\n        continue\n    idx += 1\n\nif not files:\n    raise SystemExit('no files in patch')\n\nfor file in files:\n    path = file\n    with open(path, 'r', encoding='utf-8', errors='replace') as f:\n        content = f.read().splitlines()\n    offset = 0\n    for h in hunks[file]:\n        idx = h['old_start'] - 1 + offset\n        if idx < 0 or idx > len(content):\n            raise SystemExit('hunk out of range')\n        new_lines = []\n        removed = 0\n        added = 0\n        cursor = idx\n        for l in h['lines']:\n            if l == '':\n                continue\n            prefix = l[:1]\n            text = l[1:]\n            if prefix == ' ':\n                if cursor >= len(content) or content[cursor] != text:\n                    raise SystemExit('context mismatch')\n                new_lines.append(content[cursor])\n                cursor += 1\n            elif prefix == '-':\n                if cursor >= len(content) or content[cursor] != text:\n                    raise SystemExit('delete mismatch')\n                cursor += 1\n                removed += 1\n            elif prefix == '+':\n                new_lines.append(text)\n                added += 1\n            else:\n                raise SystemExit('invalid hunk line')\n        content = content[:idx] + new_lines + content[cursor:]\n        offset += added - removed\n    with open(path, 'w', encoding='utf-8') as f:\n        f.write('\\n'.join(content) + '\\n')\nPY\nelse echo 'patch not installed and python unavailable' >&2; exit 2; fi",
			delimiter, args.Patch, delimiter)
		r, err := h.Session.capsule.Exec(ctx, h.Session.handle, cmd, capsule.ExecOptions{Cwd: "/workspace"})
		res.Output = truncateOutput(r.Output)
		res.ExitCode = r.ExitCode
		if err != nil {
			res.Error = err.Error()
		}
		h.recordToolResult(res)
		h.emitToolOutput(action, res)
		return res
	default:
		res.Error = "unsupported tool"
		return res
	}
}

func (h SessionHandler) recordToolResult(res toolResult) {
	if h.Session == nil {
		return
	}
	h.Session.AddToolResult(toolResultSummary(res))
}

func toolResultSummary(res toolResult) string {
	out := res.Output
	if len(out) > 200 {
		out = out[:200] + "…"
	}
	err := res.Error
	if len(err) > 200 {
		err = err[:200] + "…"
	}
	if err != "" {
		return fmt.Sprintf("%s exit=%d error=%s", res.Tool, res.ExitCode, err)
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Sprintf("%s exit=%d output=%s", res.Tool, res.ExitCode, strings.TrimSpace(out))
	}
	return fmt.Sprintf("%s exit=%d", res.Tool, res.ExitCode)
}

func workspacePath(p string) (string, error) {
	if p == "" {
		return "/workspace", nil
	}
	clean := p
	if strings.HasPrefix(p, "/") {
		clean = path.Clean(p)
	} else {
		clean = path.Clean(path.Join("/workspace", p))
	}
	if clean == "/workspace" || strings.HasPrefix(clean, "/workspace/") {
		return clean, nil
	}
	return "", fmt.Errorf("path must be within /workspace")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func truncateOutput(out string) string {
	if len(out) <= toolOutputLimit {
		return out
	}
	return out[:toolOutputLimit] + "\n...[truncated]"
}

func (h SessionHandler) emitToolOutput(action protocol.Action, res toolResult) {
	if h.Session == nil {
		return
	}
	out := formatToolBlock(res)
	payload, _ := protocol.MarshalPayload(protocol.TerminalOutputPayload{
		Stream: "stdout",
		Data:   out,
	})
	h.Session.Emit(protocol.Event{
		EventID:   "tool-output",
		SessionID: action.SessionID,
		Timestamp: time.Now().UTC(),
		Type:      protocol.EventTerminalOutput,
		Source:    protocol.SourceExecutor,
		AgentID:   action.AgentID,
		Payload:   payload,
	})
}

func formatToolBlock(res toolResult) string {
	var out strings.Builder
	header := toolHeader(res)
	if header == "" {
		return ""
	}
	out.WriteString("\n")
	out.WriteString(header)
	out.WriteString("\n\n")
	if res.Error != "" {
		out.WriteString("  error: " + res.Error + "\n")
	}
	if strings.TrimSpace(res.Output) != "" {
		for _, line := range strings.Split(res.Output, "\n") {
			if line == "" {
				out.WriteString("  \n")
				continue
			}
			out.WriteString("  " + line + "\n")
		}
	}
	return out.String()
}

func toolHeader(res toolResult) string {
	switch res.Tool {
	case "shell":
		var args struct {
			Command string            `json:"command"`
			Cwd     string            `json:"cwd,omitempty"`
			Env     map[string]string `json:"env,omitempty"`
		}
		if err := json.Unmarshal([]byte(res.Args), &args); err == nil && strings.TrimSpace(args.Command) != "" {
			if args.Cwd != "" {
				return fmt.Sprintf("◆ run %s  (cwd: %s)", args.Command, args.Cwd)
			}
			return fmt.Sprintf("◆ run %s", args.Command)
		}
		return "◆ run shell command"
	case "read_file":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(res.Args), &args); err == nil && args.Path != "" {
			return fmt.Sprintf("◆ read %s", args.Path)
		}
		return "◆ read file"
	case "write_file":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(res.Args), &args); err == nil && args.Path != "" {
			return fmt.Sprintf("◆ write %s", args.Path)
		}
		return "◆ write file"
	case "list_files":
		return "◆ list files"
	case "search":
		var args struct {
			Pattern string `json:"pattern"`
			Path    string `json:"path,omitempty"`
		}
		if err := json.Unmarshal([]byte(res.Args), &args); err == nil && args.Pattern != "" {
			if args.Path != "" {
				return fmt.Sprintf("◆ search %q in %s", args.Pattern, args.Path)
			}
			return fmt.Sprintf("◆ search %q", args.Pattern)
		}
		return "◆ search"
	case "apply_patch":
		return "◆ apply patch"
	default:
		return ""
	}
}

func pickDelimiter(content string) string {
	base := "KRELLIN_PATCH_EOF_"
	for i := 0; i < 1000; i++ {
		d := base + strconv.Itoa(i)
		if !strings.Contains(content, d) {
			return d
		}
	}
	return base + "X"
}
