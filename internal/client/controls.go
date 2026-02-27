package client

// ViewMode controls which panes are visible.
type ViewMode int

const (
	ViewAll ViewMode = iota
	ViewTerminal
	ViewDiff
)

func toggleMode(current ViewMode) ViewMode {
	switch current {
	case ViewAll:
		return ViewTerminal
	case ViewTerminal:
		return ViewDiff
	default:
		return ViewAll
	}
}

func clampLines(lines []string, max int) []string {
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}
