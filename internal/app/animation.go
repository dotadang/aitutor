package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/naorpeled/aitutor/internal/ui"
)

type animTickMsg time.Time

func animTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

// neuralNet renders an animated neural network ASCII art.
type neuralNet struct {
	frame int
}

func (n *neuralNet) advance() {
	n.frame++
}

func (n neuralNet) View() string {
	accent := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
	dim := lipgloss.NewStyle().Foreground(ui.ColorDim)
	green := lipgloss.NewStyle().Foreground(ui.ColorBeginner)
	yellow := lipgloss.NewStyle().Foreground(ui.ColorIntermediate)
	highlight := lipgloss.NewStyle().Foreground(ui.ColorHighlight)

	styles := []lipgloss.Style{dim, accent, green, yellow, highlight}
	pick := func(nodeIdx int) lipgloss.Style {
		phase := (n.frame + nodeIdx) % (len(styles) * 2)
		if phase >= len(styles) {
			phase = len(styles)*2 - phase - 1
		}
		return styles[phase]
	}

	node := func(idx int) string {
		return pick(idx).Render("o")
	}
	conn := func(idx int, ch string) string {
		return pick(idx).Render(ch)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines,
		"       "+node(0)+conn(0, "--")+conn(1, "--")+conn(2, `\`)+" "+node(4)+conn(4, "-")+conn(5, `\`),
	)
	lines = append(lines,
		"      "+conn(0, "/")+`   `+conn(2, `\`)+` `+conn(3, "/")+`  `+conn(5, `\`)+"  "+node(7),
	)
	lines = append(lines,
		"     "+node(1)+"    "+node(5)+"      "+conn(7, "-")+conn(7, "-")+node(8),
	)
	lines = append(lines,
		"      "+conn(1, `\`)+`   `+conn(3, "/")+` `+conn(5, `\`)+`  `+conn(6, "/")+"  "+node(9),
	)
	lines = append(lines,
		"       "+node(2)+conn(2, "--")+conn(3, "--")+conn(3, "/")+" "+node(6)+conn(6, "-")+conn(7, "/"),
	)
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}
