package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
}

func newTUIModel(c client.Client, sessionID, agentID string) *tuiModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type a command, !command, or /agents"
	ti.Focus()
	ti.CharLimit = 512
	ti.SetWidth(60)

	return &tuiModel{
		client:    c,
		sessionID: sessionID,
		agentID:   agentID,
		input:     ti,
		outputVP:  viewport.New(),
		inputVP:   viewport.New(),
		showLogo:  true,
		histIdx:   -1,
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
			if !m.ready {
				m.ready = true
			}
			return m, m.sendActionCmd(line)
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
	case disconnectedMsg:
		m.ready = false
		m.appendTerminal("[disconnected] retrying...")
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
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m *tuiModel) sendActionCmd(line string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if strings.HasPrefix(line, "/agents") {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
