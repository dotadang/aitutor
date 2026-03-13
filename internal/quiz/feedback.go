package quiz

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/naorpeled/aitutor/internal/i18n"
)

var (
	correctStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ade80")).
			Bold(true)

	incorrectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f87171")).
			Bold(true)

	explanationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#818cf8")).
				PaddingLeft(2)
)

func RenderCorrect(explanation string) string {
	result := correctStyle.Render(i18n.Text("  ✓ Correct!"))
	if explanation != "" {
		result += "\n" + explanationStyle.Render(i18n.Text(explanation))
	}
	return result
}

func RenderIncorrect(explanation string) string {
	result := incorrectStyle.Render(i18n.Text("  ✗ Incorrect"))
	if explanation != "" {
		result += "\n" + explanationStyle.Render(i18n.Text(explanation))
	}
	return result
}
