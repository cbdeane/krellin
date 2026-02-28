package session

import (
	"fmt"
	"os"
	"strings"
)

const (
	maxChatTurns     = 12
	maxToolSummaries = 6
)

type chatTurn struct {
	Role    string
	Content string
}

func (s *Session) AddChatTurn(role, content string) {
	if s == nil || strings.TrimSpace(content) == "" {
		return
	}
	s.contextMu.Lock()
	defer s.contextMu.Unlock()
	s.chatHistory = append(s.chatHistory, chatTurn{Role: role, Content: content})
	if len(s.chatHistory) > maxChatTurns {
		s.chatHistory = s.chatHistory[len(s.chatHistory)-maxChatTurns:]
	}
}

func (s *Session) AddToolResult(summary string) {
	if s == nil || strings.TrimSpace(summary) == "" {
		return
	}
	s.contextMu.Lock()
	defer s.contextMu.Unlock()
	s.toolSummaries = append(s.toolSummaries, summary)
	if len(s.toolSummaries) > maxToolSummaries {
		s.toolSummaries = s.toolSummaries[len(s.toolSummaries)-maxToolSummaries:]
	}
}

func (s *Session) snapshotContext() (history []chatTurn, tools []string) {
	s.contextMu.Lock()
	defer s.contextMu.Unlock()
	history = append([]chatTurn{}, s.chatHistory...)
	tools = append([]string{}, s.toolSummaries...)
	return history, tools
}

func buildAgentPrompt(s *Session, userContent string) string {
	var out strings.Builder
	out.WriteString("System context:\n")
	out.WriteString(workspacePreamble(s))
	out.WriteString("\n")
	history, tools := s.snapshotContext()
	if len(history) > 0 {
		out.WriteString("Conversation history:\n")
		for _, turn := range history {
			out.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(turn.Role), turn.Content))
		}
		out.WriteString("\n")
	}
	if len(tools) > 0 {
		out.WriteString("Recent tool results (summary):\n")
		for _, line := range tools {
			out.WriteString("- " + line + "\n")
		}
		out.WriteString("\n")
	}
	out.WriteString("Tool availability:\n")
	out.WriteString("- You can use tools to read/write files and run shell commands inside the capsule.\n")
	out.WriteString("- For any file or system changes, you must call tools rather than describing actions.\n\n")
	out.WriteString("User:\n")
	out.WriteString(userContent)
	return out.String()
}

func requiresTools(userContent string) bool {
	lower := strings.ToLower(userContent)
	verbs := []string{"write", "create", "edit", "modify", "update", "delete", "rename", "read", "open", "cat", "diff", "patch", "apply patch"}
	objects := []string{"file", "directory", "folder", ".md", ".txt", ".go", ".js", ".ts", ".py", ".json", ".yaml", ".yml"}
	verbHit := false
	for _, v := range verbs {
		if strings.Contains(lower, v) {
			verbHit = true
			break
		}
	}
	if !verbHit {
		return false
	}
	for _, o := range objects {
		if strings.Contains(lower, o) {
			return true
		}
	}
	return false
}

func requiresWrite(userContent string) bool {
	lower := strings.ToLower(userContent)
	verbs := []string{"write", "create", "edit", "modify", "update", "delete", "rename", "patch", "apply patch", "add", "append", "extend", "expand", "continue", "insert", "replace", "change", "fix"}
	objects := []string{"file", "directory", "folder", ".md", ".txt", ".go", ".js", ".ts", ".py", ".json", ".yaml", ".yml"}
	verbHit := false
	for _, v := range verbs {
		if strings.Contains(lower, v) {
			verbHit = true
			break
		}
	}
	if !verbHit {
		return false
	}
	for _, o := range objects {
		if strings.Contains(lower, o) {
			return true
		}
	}
	return false
}

func requiresMultiStep(userContent string) bool {
	lower := strings.ToLower(userContent)
	if strings.Contains(lower, " and then ") || strings.Contains(lower, " then ") {
		return true
	}
	return false
}

func workspacePreamble(s *Session) string {
	if s == nil {
		return ""
	}
	root := s.repoRoot
	if root == "" {
		root = "."
	}
	tree := topLevelTree(root)
	lines := []string{
		fmt.Sprintf("repo_root: %s", root),
		fmt.Sprintf("capsule_image: %s", s.imageDigest),
		fmt.Sprintf("capsule_user: %s", orDefault(s.capsuleUser, "default")),
		fmt.Sprintf("network: %t", s.networkOn),
	}
	if tree != "" {
		lines = append(lines, "top_level: "+tree)
	}
	return strings.Join(lines, "\n")
}

func topLevelTree(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	names := []string{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			name = name + "/"
		}
		names = append(names, name)
		if len(names) >= 20 {
			break
		}
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, " ")
}

func orDefault(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}
