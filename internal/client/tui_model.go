package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"krellin/internal/protocol"
	"krellin/pkg/client"
)

type eventMsg struct {
	ev protocol.Event
}

type errMsg struct {
	err error
}

type disconnectedMsg struct{}
type connectedMsg struct{}

type localCmdResultMsg struct {
	output string
	err    error
}

type tuiModel struct {
	client    client.Client
	sessionID string
	agentID   string

	width  int
	height int

	terminal []string
	input    textinput.Model
	outputVP viewport.Model
	inputVP  viewport.Model
	ready    bool
	showLogo bool
	history  []string
	histIdx  int
	lastEsc  time.Time

	actionLog []string
	lastDiff  string

	agentsOpen      bool
	agentsLoading   bool
	agentsProviders []protocol.AgentProviderInfo
	agentsActive    string
	agentsSelected  int

	agentsMode        string
	agentsAddFields   []textinput.Model
	agentsAddEnabled  bool
	agentsAddFieldIdx int
	agentsAddErr      string
	agentsAddMode     string

	localRunner func(string) (string, error)

	agentsDeletePending bool
	agentsDeleteName    string
}

func newTUIModel(c client.Client, sessionID, agentID string) *tuiModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type a command, !command, or /agents"
	ti.Focus()
	ti.CharLimit = 512
	ti.SetWidth(60)

	return &tuiModel{
		client:      c,
		sessionID:   sessionID,
		agentID:     agentID,
		input:       ti,
		outputVP:    viewport.New(),
		inputVP:     viewport.New(),
		showLogo:    true,
		histIdx:     -1,
		localRunner: defaultLocalRunner,
	}
}

func (m *tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		if m.agentsOpen {
			if m.agentsMode == "add" {
				switch msg.String() {
				case "esc":
					m.agentsMode = "list"
					return m, nil
				case "tab", "down":
					m.setAgentsAddField(m.agentsAddFieldIdx + 1)
					return m, nil
				case "shift+tab", "up":
					m.setAgentsAddField(m.agentsAddFieldIdx - 1)
					return m, nil
				case "ctrl+s":
					return m, m.submitAgentsAddCmd()
				case "enter":
					if m.agentsAddFieldIdx < len(m.agentsAddFields) {
						m.setAgentsAddField(m.agentsAddFieldIdx + 1)
						return m, nil
					}
					return m, m.submitAgentsAddCmd()
				case " ":
					if m.agentsAddFieldIdx == len(m.agentsAddFields) {
						m.agentsAddEnabled = !m.agentsAddEnabled
						return m, nil
					}
				}
				if m.agentsAddFieldIdx < len(m.agentsAddFields) {
					var cmd tea.Cmd
					idx := m.agentsAddFieldIdx
					m.agentsAddFields[idx], cmd = m.agentsAddFields[idx].Update(msg)
					return m, cmd
				}
				return m, nil
			}
			switch msg.String() {
			case "esc":
				m.agentsOpen = false
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				return m, nil
			case "up":
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				if m.agentsSelected > 0 {
					m.agentsSelected--
				}
				return m, nil
			case "down":
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				if m.agentsSelected < len(m.agentsProviders)-1 {
					m.agentsSelected++
				}
				return m, nil
			case "enter":
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				if len(m.agentsProviders) == 0 {
					return m, nil
				}
				name := m.agentsProviders[m.agentsSelected].Name
				return m, m.sendAgentsSetActiveCmd(name)
			case " ":
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				if len(m.agentsProviders) == 0 {
					return m, nil
				}
				prov := m.agentsProviders[m.agentsSelected]
				return m, m.sendAgentsToggleCmd(prov.Name, !prov.Enabled)
			case "a":
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				m.openAgentsAdd()
				return m, nil
			case "e":
				m.agentsDeletePending = false
				m.agentsDeleteName = ""
				if len(m.agentsProviders) == 0 {
					return m, nil
				}
				m.openAgentsEdit(m.agentsProviders[m.agentsSelected])
				return m, nil
			case "d":
				if len(m.agentsProviders) == 0 {
					return m, nil
				}
				name := m.agentsProviders[m.agentsSelected].Name
				if m.agentsDeletePending && m.agentsDeleteName == name {
					m.agentsDeletePending = false
					m.agentsDeleteName = ""
					return m, m.sendAgentsDeleteCmd(name)
				}
				m.agentsDeletePending = true
				m.agentsDeleteName = name
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			now := time.Now()
			if !m.lastEsc.IsZero() && now.Sub(m.lastEsc) < 500*time.Millisecond {
				m.input.SetValue("")
				m.input.CursorStart()
				m.lastEsc = time.Time{}
				m.updateOutput()
				return m, nil
			}
			m.lastEsc = now
			return m, nil
		case "up":
			if m.input.Position() == 0 && len(m.history) > 0 {
				if m.histIdx < 0 {
					m.histIdx = len(m.history) - 1
				} else if m.histIdx > 0 {
					m.histIdx--
				}
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
				m.updateOutput()
				return m, nil
			}
		case "enter":
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if line == "" {
				return m, nil
			}
			m.histIdx = -1
			if line == "/exit" || line == "/quit" {
				return m, tea.Quit
			}
			if line == "/clear" || line == "clear" {
				m.terminal = nil
				m.showLogo = false
				m.updateOutput()
				return m, nil
			}
			if line == "/log" {
				if len(m.actionLog) == 0 {
					m.appendTerminal("[log] no actions yet")
					return m, nil
				}
				m.appendTerminal("[log] recent actions")
				for _, entry := range m.actionLog {
					m.appendTerminal("  " + entry)
				}
				return m, nil
			}
			if line == "/diff" {
				if m.lastDiff == "" {
					m.appendTerminal("[diff] no diff available")
					return m, nil
				}
				diff := m.lastDiff
				if !strings.HasSuffix(diff, "\n") {
					diff += "\n"
				}
				m.appendTerminal("[diff]\n" + diff)
				return m, nil
			}
			m.history = append(m.history, line)
			if m.showLogo {
				m.terminal = nil
				m.showLogo = false
				m.updateOutput()
			}
			if line == "/agents" {
				m.agentsOpen = true
				m.agentsLoading = true
				m.agentsSelected = 0
				m.agentsMode = "list"
				return m, m.sendAgentsListCmd()
			}
			if !m.ready {
				m.ready = true
			}
			return m, m.dispatchInputCmd(line)
		}
	case eventMsg:
		if !m.ready {
			m.ready = true
			m.applyEvent(msg.ev)
			return m, nil
		}
		m.applyEvent(msg.ev)
		return m, nil
	case connectedMsg:
		if !m.ready {
			m.ready = true
		}
		return m, nil
	case errMsg:
		m.appendTerminal("[error] " + msg.err.Error())
		return m, nil
	case localCmdResultMsg:
		if msg.err != nil {
			m.appendTerminal("[error] " + msg.err.Error())
			return m, nil
		}
		m.appendTerminal(msg.output)
		return m, nil
	case disconnectedMsg:
		m.ready = false
		m.appendTerminal("[disconnected] retrying...")
		return m, nil
	case tea.PasteMsg:
		if m.agentsOpen && m.agentsMode == "add" {
			if m.agentsAddFieldIdx < len(m.agentsAddFields) {
				field := m.agentsAddFields[m.agentsAddFieldIdx]
				field.SetValue(field.Value() + msg.Content)
				field.CursorEnd()
				m.agentsAddFields[m.agentsAddFieldIdx] = field
			}
			return m, nil
		}
		m.input.SetValue(m.input.Value() + msg.Content)
		m.input.CursorEnd()
		m.updateOutput()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *tuiModel) View() tea.View {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 28
	}
	// Avoid last-column/row auto-wrap in terminals.
	if width > 1 {
		width = width - 1
	}
	if height > 1 {
		height = height - 1
	}

	outputBox := m.renderOutput(width, height-4)
	inputBox := m.renderInput(width, 4)
	content := lipgloss.JoinVertical(lipgloss.Left, outputBox, inputBox)
	if m.agentsOpen {
		modal := m.renderAgentsModal(width, height)
		content = overlayCentered(content, modal, width, height)
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m *tuiModel) sendActionCmd(line string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if strings.HasPrefix(line, "!") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
			if line == "" {
				return nil
			}
		}
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionRunCommand,
			Timestamp: time.Now(),
			Payload:   encodeJSON(protocol.RunCommandPayload{Command: line, Cwd: "/workspace"}),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) dispatchInputCmd(line string) tea.Cmd {
	if strings.HasPrefix(line, "/") {
		return nil
	}
	if strings.HasPrefix(line, "!") {
		if isLocalGitCommand(line) {
			return m.runLocalCmd(line)
		}
		return m.sendActionCmd(line)
	}
	return m.sendAgentPromptCmd(line)
}

func (m *tuiModel) runLocalCmd(line string) tea.Cmd {
	return func() tea.Msg {
		cmd := strings.TrimSpace(strings.TrimPrefix(line, "!"))
		if cmd == "" {
			return nil
		}
		out, err := m.localRunner(cmd)
		return localCmdResultMsg{output: out, err: err}
	}
}

func (m *tuiModel) sendAgentPromptCmd(text string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionAgentPrompt,
			Timestamp: time.Now(),
			Payload:   encodeJSON(protocol.AgentPromptPayload{Content: text}),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) sendAgentsListCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionAgentsList,
			Timestamp: time.Now(),
			Payload:   encodeJSON(struct{}{}),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) sendAgentsSetActiveCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionAgentsSetActive,
			Timestamp: time.Now(),
			Payload:   encodeJSON(protocol.AgentsSetActivePayload{Name: name}),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) sendAgentsToggleCmd(name string, enabled bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionAgentsToggle,
			Timestamp: time.Now(),
			Payload:   encodeJSON(protocol.AgentsTogglePayload{Name: name, Enabled: enabled}),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) sendAgentsAddCmd(payload protocol.AgentsAddPayload) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionAgentsAdd,
			Timestamp: time.Now(),
			Payload:   encodeJSON(payload),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) sendAgentsDeleteCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		action := protocol.Action{
			ActionID:  "local",
			SessionID: m.sessionID,
			AgentID:   m.agentID,
			Type:      protocol.ActionAgentsDelete,
			Timestamp: time.Now(),
			Payload:   encodeJSON(protocol.AgentsDeletePayload{Name: name}),
		}
		if err := m.client.SendAction(ctx, encodeJSON(action)); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *tuiModel) applyEvent(ev protocol.Event) {
	switch ev.Type {
	case protocol.EventTerminalOutput:
		var payload protocol.TerminalOutputPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.appendTerminal(payload.Data)
		}
	case protocol.EventError:
		var payload protocol.ErrorPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.appendTerminal("[error] " + payload.Message)
		}
	case protocol.EventActionStarted:
		var payload protocol.ActionStartedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.appendActionLog(fmt.Sprintf("%s started %s (%s)", formatEventTime(ev.Timestamp), payload.ActionID, payload.Type))
		}
	case protocol.EventActionFinished:
		var payload protocol.ActionFinishedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.appendActionLog(fmt.Sprintf("%s finished %s (%s)", formatEventTime(ev.Timestamp), payload.ActionID, payload.Status))
			if payload.Status == "failure" {
				m.appendTerminal("[error] action failed: " + payload.Error)
			}
		}
	case protocol.EventAgentMessage:
		var payload protocol.AgentMessagePayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.appendTerminal("[agent] " + payload.Content)
		}
	case protocol.EventAgentsList:
		var payload protocol.AgentsListPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.agentsProviders = payload.Providers
			m.agentsActive = payload.Active
			m.agentsLoading = false
			if m.agentsSelected >= len(m.agentsProviders) {
				m.agentsSelected = maxInt(0, len(m.agentsProviders)-1)
			}
		}
	case protocol.EventDiffReady:
		var payload protocol.DiffReadyPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			m.lastDiff = payload.Patch
			m.appendActionLog(fmt.Sprintf("%s diff ready (%d files)", formatEventTime(ev.Timestamp), len(payload.Files)))
		}
	}
}

func (m *tuiModel) appendTerminal(data string) {
	if data == "" {
		return
	}
	parts := strings.Split(data, "\n")
	if len(m.terminal) == 0 {
		m.terminal = append(m.terminal, parts[0])
	} else {
		m.terminal[len(m.terminal)-1] += parts[0]
	}
	for _, part := range parts[1:] {
		m.terminal = append(m.terminal, part)
	}
	m.terminal = clampLines(m.terminal, 1000)
	m.updateOutput()
}

func (m *tuiModel) appendActionLog(entry string) {
	m.actionLog = append(m.actionLog, entry)
	if len(m.actionLog) > 50 {
		m.actionLog = m.actionLog[len(m.actionLog)-50:]
	}
}

func formatEventTime(ts time.Time) string {
	if ts.IsZero() {
		return "--:--:--"
	}
	return ts.Local().Format("15:04:05")
}

func (m *tuiModel) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	width := m.width
	height := m.height
	if width > 1 {
		width = width - 1
	}
	if height > 1 {
		height = height - 1
	}
	outputHeight := height - 4
	if outputHeight < 3 {
		outputHeight = 3
	}
	m.outputVP.SetWidth(maxInt(10, width-2))
	m.outputVP.SetHeight(maxInt(1, outputHeight-2))
	m.input.SetWidth(maxInt(10, width-6))
	m.inputVP.SetWidth(maxInt(10, width-2))
	m.inputVP.SetHeight(2)
	m.outputVP.SoftWrap = true
	m.inputVP.SoftWrap = true
	m.updateOutput()
}

func (m *tuiModel) updateOutput() {
	if m.outputVP.Width() <= 0 {
		return
	}
	m.outputVP.SetContent(strings.Join(m.terminal, "\n"))
	m.inputVP.SetContent(m.input.View())
}

func (m *tuiModel) renderOutput(width, height int) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3A6F7A")).
		Foreground(lipgloss.Color("#E6F1F5")).
		Padding(0, 1)
	m.outputVP.Style = style
	m.outputVP.SetWidth(width)
	m.outputVP.SetHeight(height)
	if m.showLogo && len(m.terminal) == 0 {
		m.outputVP.SetContent(strings.Join(logoLines(), "\n"))
	} else {
		m.outputVP.SetContent(strings.Join(m.terminal, "\n"))
	}
	return m.outputVP.View()
}

func (m *tuiModel) renderInput(width, height int) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3A6F7A")).
		Foreground(lipgloss.Color("#E6F1F5")).
		Padding(0, 1)
	m.inputVP.Style = style
	m.inputVP.SetWidth(width)
	m.inputVP.SetHeight(height)
	m.inputVP.SetContent(m.input.View())
	return m.inputVP.View()
}

func (m *tuiModel) renderAgentsModal(width, height int) string {
	if m.agentsMode == "add" {
		return m.renderAgentsAddModal(width, height)
	}
	return m.renderAgentsListModal(width, height)
}

func (m *tuiModel) renderAgentsListModal(width, height int) string {
	modalWidth := minInt(80, maxInt(50, width-8))
	modalHeight := minInt(22, maxInt(10, height-6))

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4D06F")).Render("Agents")
	header := title + "\n"

	lines := []string{}
	if m.agentsLoading {
		lines = append(lines, "Loading providers...")
	} else if len(m.agentsProviders) == 0 {
		lines = append(lines, "No providers configured.")
		lines = append(lines, "Press 'a' to add one.")
	} else {
		for i, prov := range m.agentsProviders {
			cursor := "  "
			if i == m.agentsSelected {
				cursor = "› "
			}
			active := ""
			if prov.Name == m.agentsActive {
				active = " (active)"
			}
			keyStatus := ""
			if prov.HasAPIKey {
				keyStatus = " key=stored"
			}
			line := fmt.Sprintf("%s%s [%s] %s — %s%s%s", cursor, prov.Name, prov.Type, prov.Model, prov.Status, keyStatus, active)
			lines = append(lines, line)
		}
	}

	footer := "\n↑/↓ select  space toggle  enter set active  a add  e edit  d delete  esc close"
	if m.agentsDeletePending {
		footer = fmt.Sprintf("\nConfirm delete %q: press d again (esc to cancel)", m.agentsDeleteName)
	}
	body := strings.Join(lines, "\n")
	content := header + body + footer

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F4D06F")).
		Foreground(lipgloss.Color("#F7F7F2")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)
	return style.Render(content)
}

func (m *tuiModel) renderAgentsAddModal(width, height int) string {
	modalWidth := minInt(80, maxInt(50, width-8))
	modalHeight := minInt(22, maxInt(12, height-6))
	fieldWidth := maxInt(20, modalWidth-20)

	titleText := "Add Provider"
	if m.agentsAddMode == "edit" {
		titleText = "Edit Provider"
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4D06F")).Render(titleText)
	header := title + "\n"

	for i := range m.agentsAddFields {
		m.agentsAddFields[i].SetWidth(fieldWidth)
	}

	lines := []string{}
	labels := []string{"Name", "Type", "Model", "API key", "API key env", "Base URL"}
	for i, label := range labels {
		cursor := "  "
		if i == m.agentsAddFieldIdx {
			cursor = "› "
		}
		line := fmt.Sprintf("%s%s: %s", cursor, label, m.agentsAddFields[i].View())
		lines = append(lines, line)
	}
	enabledCursor := "  "
	if m.agentsAddFieldIdx == len(m.agentsAddFields) {
		enabledCursor = "› "
	}
	enabledMark := "[ ]"
	if m.agentsAddEnabled {
		enabledMark = "[x]"
	}
	lines = append(lines, fmt.Sprintf("%sEnabled: %s", enabledCursor, enabledMark))

	if hint := m.agentsAddHint(); hint != "" {
		hintLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#A0C4FF")).Render(hint)
		lines = append(lines, hintLine)
	}
	if m.agentsAddErr != "" {
		errLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#F25C54")).Render(m.agentsAddErr)
		lines = append(lines, errLine)
	}

	footer := "\nenter next  ctrl+s save  esc back"
	body := strings.Join(lines, "\n")
	content := header + body + footer

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F4D06F")).
		Foreground(lipgloss.Color("#F7F7F2")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)
	return style.Render(content)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *tuiModel) openAgentsAdd() {
	m.agentsMode = "add"
	m.agentsAddMode = "add"
	m.agentsAddErr = ""
	m.agentsAddEnabled = true
	m.agentsAddFields = []textinput.Model{
		newAgentsField("provider-name"),
		newAgentsField("openai|anthropic|grok|gemini|llama"),
		newAgentsField("model-id"),
		newAgentsField("paste key (optional)"),
		newAgentsField("API_KEY_ENV"),
		newAgentsField("https://... (optional)"),
	}
	m.setAgentsAddField(0)
}

func (m *tuiModel) openAgentsEdit(provider protocol.AgentProviderInfo) {
	m.agentsMode = "add"
	m.agentsAddMode = "edit"
	m.agentsAddErr = ""
	m.agentsAddEnabled = provider.Enabled
	m.agentsAddFields = []textinput.Model{
		newAgentsField("provider-name"),
		newAgentsField("openai|anthropic|grok|gemini|llama"),
		newAgentsField("model-id"),
		newAgentsField("paste key (optional)"),
		newAgentsField("API_KEY_ENV"),
		newAgentsField("https://... (optional)"),
	}
	m.agentsAddFields[0].SetValue(provider.Name)
	m.agentsAddFields[1].SetValue(provider.Type)
	m.agentsAddFields[2].SetValue(provider.Model)
	m.agentsAddFields[4].SetValue(provider.APIKeyEnv)
	m.agentsAddFields[5].SetValue(provider.BaseURL)
	m.setAgentsAddField(0)
}

func newAgentsField(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	if placeholder == "paste key (optional)" {
		ti.EchoMode = textinput.EchoPassword
	} else {
		ti.EchoMode = textinput.EchoNormal
	}
	ti.CharLimit = 128
	return ti
}

func (m *tuiModel) setAgentsAddField(idx int) {
	if idx < 0 {
		idx = 0
	}
	maxIdx := len(m.agentsAddFields)
	if idx > maxIdx {
		idx = maxIdx
	}
	m.agentsAddFieldIdx = idx
	for i := range m.agentsAddFields {
		if i == idx {
			m.agentsAddFields[i].Focus()
		} else {
			m.agentsAddFields[i].Blur()
		}
	}
	if idx == maxIdx {
		for i := range m.agentsAddFields {
			m.agentsAddFields[i].Blur()
		}
	}
}

func (m *tuiModel) submitAgentsAddCmd() tea.Cmd {
	payload := protocol.AgentsAddPayload{
		Name:      strings.TrimSpace(m.agentsAddFields[0].Value()),
		Type:      strings.TrimSpace(m.agentsAddFields[1].Value()),
		Model:     strings.TrimSpace(m.agentsAddFields[2].Value()),
		APIKey:    strings.TrimSpace(m.agentsAddFields[3].Value()),
		APIKeyEnv: strings.TrimSpace(m.agentsAddFields[4].Value()),
		BaseURL:   strings.TrimSpace(m.agentsAddFields[5].Value()),
		Enabled:   m.agentsAddEnabled,
	}
	if payload.Name == "" || payload.Type == "" || payload.Model == "" || (payload.APIKey == "" && payload.APIKeyEnv == "") {
		m.agentsAddErr = "Name, type, model, and API key (or env) are required."
		return nil
	}
	normalized := strings.ToLower(payload.Type)
	switch normalized {
	case "openai", "anthropic", "grok", "gemini", "llama":
	default:
		m.agentsAddErr = "Type must be openai, anthropic, grok, gemini, or llama."
		return nil
	}
	payload.Type = normalized
	if payload.Type == "llama" && payload.BaseURL == "" {
		m.agentsAddErr = "Base URL is required for llama."
		return nil
	}
	m.agentsAddErr = ""
	m.agentsMode = "list"
	m.agentsLoading = true
	return m.sendAgentsAddCmd(payload)
}

func (m *tuiModel) agentsAddHint() string {
	field := m.agentsAddFieldIdx
	typ := strings.ToLower(strings.TrimSpace(m.agentsAddFields[1].Value()))
	if field == 1 {
		return "Types: openai, anthropic, grok, gemini, llama"
	}
	if field == 3 {
		return "Paste an API key to store in your system keychain"
	}
	if field == 4 {
		switch typ {
		case "openai":
			return "Suggested env: OPENAI_API_KEY"
		case "anthropic":
			return "Suggested env: ANTHROPIC_API_KEY"
		case "grok":
			return "Suggested env: GROK_API_KEY"
		case "gemini":
			return "Suggested env: GEMINI_API_KEY"
		case "llama":
			return "Suggested env: LLAMA_API_KEY"
		}
	}
	if field == 5 {
		if typ == "llama" {
			return "Required for llama (e.g., http://localhost:8000/v1)"
		}
		return "Optional for hosted providers"
	}
	if field == 2 && typ != "" {
		return "Enter a model id for the selected provider"
	}
	return ""
}

func overlayCentered(base, overlay string, width, height int) string {
	baseLines := normalizeLines(base, height)
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > overlayWidth {
			overlayWidth = w
		}
	}
	overlayHeight := len(overlayLines)
	startX := (width - overlayWidth) / 2
	startY := (height - overlayHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for i, line := range overlayLines {
		y := startY + i
		if y < 0 || y >= len(baseLines) {
			continue
		}
		baseLine := baseLines[y]
		prefix := truncateByWidth(baseLine, startX)
		suffix := trimLeftByWidth(baseLine, startX+overlayWidth)
		baseLines[y] = prefix + line + suffix
	}
	return strings.Join(baseLines, "\n")
}

func normalizeLines(content string, height int) []string {
	lines := strings.Split(content, "\n")
	if height <= 0 {
		return lines
	}
	if len(lines) < height {
		for i := len(lines); i < height; i++ {
			lines = append(lines, "")
		}
		return lines
	}
	return lines[:height]
}

func truncateByWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}

func trimLeftByWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) <= width {
		return ""
	}
	return ansi.TruncateLeft(s, width, "")
}

func isLocalGitCommand(line string) bool {
	if !strings.HasPrefix(line, "!") {
		return false
	}
	cmd := strings.TrimSpace(strings.TrimPrefix(line, "!"))
	return cmd == "git" || strings.HasPrefix(cmd, "git ")
}

func defaultLocalRunner(cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
	return string(out), err
}

func logoLines() []string {
	return []string{
		"██╗  ██╗██████╗ ███████╗██╗     ██╗     ██╗███╗   ██╗",
		"██║ ██╔╝██╔══██╗██╔════╝██║     ██║     ██║████╗  ██║",
		"█████╔╝ ██████╔╝█████╗  ██║     ██║     ██║██╔██╗ ██║",
		"██╔═██╗ ██╔══██╗██╔══╝  ██║     ██║     ██║██║╚██╗██║",
		"██║  ██╗██║  ██║███████╗███████╗███████╗██║██║ ╚████║",
		"╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝╚═╝  ╚═══╝",
		"",
		"Local capsules. Serialized actions. No git automation.",
	}
}
