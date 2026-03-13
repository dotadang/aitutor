package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/naorpeled/aitutor/internal/i18n"
	"github.com/naorpeled/aitutor/internal/lesson"
	"github.com/naorpeled/aitutor/internal/progress"
	"github.com/naorpeled/aitutor/internal/ui"
	"github.com/naorpeled/aitutor/pkg/types"
)

// AppModel is the root Bubbletea model.
type AppModel struct {
	width       int
	height      int
	layout      ui.Layout
	header      ui.HeaderModel
	footer      ui.FooterModel
	sidebarOpen bool
	ready       bool
	showWelcome bool
	showHelp    bool
	showLanguage bool
	version     string
	anim        neuralNet
	language      i18n.Language
	languageCursor int

	lessons     []types.LessonDef
	lessonIdx   int
	lessonModel lesson.Model
	sidebar     ui.SidebarModel
	tracker     *progress.Tracker
}

func NewAppModel(version string) AppModel {
	return AppModel{
		header:      ui.NewHeaderModel(),
		footer:      ui.NewFooterModel(),
		sidebarOpen: false,
		showWelcome: true,
		language:    i18n.English,
		version:     version,
	}
}

func (m AppModel) Init() tea.Cmd {
	return animTick()
}

func (m *AppModel) loadLessons() {
	m.lessons = lesson.All()
	if len(m.lessons) > 0 {
		m.header.Total = len(m.lessons)
		m.sidebar = ui.NewSidebarModel()
		m.sidebar.Lessons = m.lessons
		m.tracker = progress.NewTracker(len(m.lessons))
		m.language = i18n.NormalizeLanguage(m.tracker.Language())
		i18n.SetLanguage(m.language)
		m.sidebar.Completed = m.tracker.CompletedMap()
		m.syncLanguageCursor()

		// Resume from last lesson
		startIdx := m.tracker.LastLessonIdx()
		if startIdx >= len(m.lessons) {
			startIdx = 0
		}
		m.selectLesson(startIdx)
	}
}

func (m *AppModel) selectLesson(idx int) {
	if idx < 0 || idx >= len(m.lessons) {
		return
	}
	m.lessonIdx = idx
	def := m.lessons[idx]
	m.header.Tier = def.Tier
	m.header.LessonTitle = def.Title
	m.header.Current = idx + 1
	m.sidebar.Active = idx
	m.lessonModel = lesson.New(def, m.layout.ContentWidth-2, m.layout.ContentHeight-2)
	m.lessonModel.IsLast = idx == len(m.lessons)-1

	if m.tracker != nil {
		m.tracker.SetLastLesson(idx)
	}
}

func (m *AppModel) syncLanguageCursor() {
	for idx, lang := range i18n.Languages() {
		if lang == m.language {
			m.languageCursor = idx
			return
		}
	}
	m.languageCursor = 0
}

func (m *AppModel) applyLanguage(lang i18n.Language) {
	m.language = i18n.NormalizeLanguage(string(lang))
	i18n.SetLanguage(m.language)
	m.syncLanguageCursor()
	if m.tracker != nil {
		m.tracker.SetLanguage(string(m.language))
	}
}

func (m *AppModel) markLessonComplete() {
	if m.tracker != nil && m.lessonIdx < len(m.lessons) {
		m.tracker.CompleteLesson(m.lessons[m.lessonIdx].ID)
		m.sidebar.Completed = m.tracker.CompletedMap()
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case animTickMsg:
		if m.showWelcome {
			m.anim.advance()
			return m, animTick()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = ui.ComputeLayout(m.width, m.height, m.sidebarOpen)
		m.header.Width = m.width
		m.footer.Width = m.width

		if !m.ready {
			m.ready = true
			m.loadLessons()
		} else {
			lm, cmd := m.lessonModel.Update(tea.WindowSizeMsg{
				Width:  m.layout.ContentWidth - 2,
				Height: m.layout.ContentHeight - 2,
			})
			m.lessonModel = lm
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Welcome screen: any key dismisses
		if m.showWelcome {
			if key.Matches(msg, Keys.Quit) {
				return m, tea.Quit
			}
			if key.Matches(msg, Keys.Language) {
				m.showLanguage = true
				return m, nil
			}
			if m.showLanguage {
				return m.updateLanguageSelector(msg)
			}
			m.showWelcome = false
			return m, nil
		}

		// Help overlay: any key dismisses
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		if m.showLanguage {
			return m.updateLanguageSelector(msg)
		}

		switch {
		case key.Matches(msg, Keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, Keys.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, Keys.Language):
			m.showLanguage = true
			return m, nil
		case key.Matches(msg, Keys.Tab):
			m.sidebarOpen = !m.sidebarOpen
			m.layout = ui.ComputeLayout(m.width, m.height, m.sidebarOpen)
			return m, nil
		case key.Matches(msg, Keys.Next):
			if m.lessonIdx < len(m.lessons)-1 {
				m.selectLesson(m.lessonIdx + 1)
			}
			return m, nil
		case key.Matches(msg, Keys.Prev):
			if m.lessonIdx > 0 {
				m.selectLesson(m.lessonIdx - 1)
			}
			return m, nil
		case key.Matches(msg, Keys.AdvancePhase):
			// Right arrow always advances the phase
			prevPhase := m.lessonModel.Phase
			m.lessonModel.Advance()
			if prevPhase != lesson.PhaseComplete && m.lessonModel.Phase == lesson.PhaseComplete {
				m.markLessonComplete()
			}
			return m, nil
		case key.Matches(msg, Keys.Advance):
			phase := m.lessonModel.Phase
			if phase == lesson.PhaseTheory {
				m.lessonModel.Advance()
				return m, nil
			}
			if phase == lesson.PhaseComplete {
				// Already complete, do nothing on Enter
				return m, nil
			}
			// Fall through to forward to lesson model (viz/quiz)
		case key.Matches(msg, Keys.Back):
			m.lessonModel.GoBack()
			return m, nil
		}
	}

	// Forward to lesson model (handles viz/quiz interactions)
	prevPhase := m.lessonModel.Phase
	var cmd tea.Cmd
	m.lessonModel, cmd = m.lessonModel.Update(msg)

	// Check if lesson just completed
	if prevPhase != lesson.PhaseComplete && m.lessonModel.Phase == lesson.PhaseComplete {
		m.markLessonComplete()
	}

	return m, cmd
}

func (m AppModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.showWelcome {
		return m.viewWelcome()
	}

	if m.showLanguage {
		return m.viewLanguageSelector()
	}

	if m.showHelp {
		return m.viewHelp()
	}

	// Show course completion screen when all lessons done and on last lesson's complete phase
	if m.tracker != nil && m.tracker.CompletedCount() >= len(m.lessons) &&
		m.lessonModel.Phase == lesson.PhaseComplete {
		return m.viewCourseComplete()
	}

	// Update footer hints based on lesson phase
	switch m.lessonModel.Phase {
	case lesson.PhaseTheory:
		m.footer.Bindings = []ui.KeyHint{
			{Key: "q", Desc: "quit"}, {Key: "Tab", Desc: "sidebar"}, {Key: "n/p", Desc: "next/prev lesson"},
			{Key: "→/Enter", Desc: "next phase"}, {Key: "↑/↓", Desc: "scroll"}, {Key: "l", Desc: "language"},
		}
	case lesson.PhaseViz:
		m.footer.Bindings = []ui.KeyHint{
			{Key: "q", Desc: "quit"}, {Key: "Tab", Desc: "sidebar"}, {Key: "n/p", Desc: "next/prev lesson"},
			{Key: "←/→", Desc: "prev/next phase"}, {Key: "Enter/Space", Desc: "interact"}, {Key: "l", Desc: "language"},
		}
	case lesson.PhaseQuiz:
		m.footer.Bindings = []ui.KeyHint{
			{Key: "q", Desc: "quit"}, {Key: "Tab", Desc: "sidebar"}, {Key: "n/p", Desc: "next/prev lesson"},
			{Key: "←", Desc: "prev phase"}, {Key: "1-4", Desc: "answer"}, {Key: "l", Desc: "language"},
		}
	case lesson.PhaseComplete:
		m.footer.Bindings = []ui.KeyHint{
			{Key: "q", Desc: "quit"}, {Key: "Tab", Desc: "sidebar"}, {Key: "n", Desc: "next lesson"},
			{Key: "←", Desc: "prev phase"}, {Key: "l", Desc: "language"},
		}
	}

	// Progress bar in header
	completedCount := 0
	if m.tracker != nil {
		completedCount = m.tracker.CompletedCount()
	}
	progressStr := progress.Bar(completedCount, len(m.lessons), 30)

	header := m.header.ViewWithProgress(progressStr)

	// Content
	contentWidth := m.layout.ContentWidth
	contentHeight := m.layout.ContentHeight
	content := ui.ContentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(m.lessonModel.View())

	// Sidebar
	var body string
	if m.layout.SidebarOpen {
		m.sidebar.Width = m.layout.SidebarWidth
		m.sidebar.Height = contentHeight
		sidebar := m.sidebar.View()
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	} else {
		body = content
	}

	footer := m.footer.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m AppModel) viewWelcome() string {
	accent := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
	bright := lipgloss.NewStyle().Foreground(ui.ColorBright)
	dim := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	green := lipgloss.NewStyle().Foreground(ui.ColorBeginner)
	yellow := lipgloss.NewStyle().Foreground(ui.ColorIntermediate)
	red := lipgloss.NewStyle().Foreground(ui.ColorAdvanced)

	logo := accent.Render(`
     _    ___ _____      _
    / \  |_ _|_   _|   _| |_ ___  _ __
   / _ \  | |  | || | | | __/ _ \| '__|
  / ___ \ | |  | || |_| | || (_) | |
 /_/   \_\___| |_| \__,_|\__\___/|_|`)

	var lines []string
	// Only show animation if terminal is tall enough (animation adds ~8 lines)
	if m.height >= 35 {
		lines = append(lines, m.anim.View())
	}
	lines = append(lines, logo)
	lines = append(lines, "")
	tagline := i18n.Text("Interactive AI Coding Concepts Tutorial")
	visibleLen := m.anim.frame * 2
	if visibleLen > len(tagline) {
		visibleLen = len(tagline)
	}
	lines = append(lines, bright.Render("  "+tagline[:visibleLen]))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Learn AI-assisted development through hands-on lessons.")))
	lines = append(lines, dim.Render(i18n.Text("  Each lesson has theory, an interactive visualization, and a quiz.")))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  Beginner      %s",
		green.Render("*"), dim.Render(i18n.Text("Context windows, tools, prompts"))))
	lines = append(lines, fmt.Sprintf("  %s  Intermediate  %s",
		yellow.Render("*"), dim.Render(i18n.Text("CLAUDE.md, hooks, memory, modes"))))
	lines = append(lines, fmt.Sprintf("  %s  Advanced      %s",
		red.Render("*"), dim.Render(i18n.Text("MCP, skills, subagents, worktrees"))))
	lines = append(lines, "")

	completedCount := 0
	if m.tracker != nil {
		completedCount = m.tracker.CompletedCount()
	}
	if completedCount > 0 {
		lines = append(lines, green.Render(i18n.Textf("  Progress: %d/%d lessons completed", completedCount, len(m.lessons))))
		lines = append(lines, "")
	}

	lines = append(lines, accent.Render(i18n.Text("  Press any key to start")))
	lines = append(lines, dim.Render(i18n.Text("  Press q to quit")))
	lines = append(lines, dim.Render(i18n.Textf("  Current language: %s", i18n.LanguageName(m.language))))
	lines = append(lines, dim.Render("  [l] " + i18n.Text("Open language selector")))
	lines = append(lines, "")
	lines = append(lines, dim.Render("  "+m.version))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Contribute → github.com/naorpeled/aitutor")))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Content is community-contributed and may be AI-assisted.")))
	lines = append(lines, dim.Render(i18n.Text("  It may contain errors. Not a substitute for professional")))
	lines = append(lines, dim.Render(i18n.Text("  training. Contributions and corrections are welcome.")))

	content := strings.Join(lines, "\n")

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content)
}

func (m AppModel) viewHelp() string {
	accent := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
	bright := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	dim := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Bold(true).Width(16)

	var lines []string
	lines = append(lines, accent.Render(i18n.Text("  Help")))
	lines = append(lines, "")
	lines = append(lines, bright.Render(i18n.Text("  Navigation")))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("Tab"), dim.Render(i18n.Text("Toggle sidebar"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("n / p"), dim.Render(i18n.Text("Next / previous lesson"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("Up/Down  j/k"), dim.Render(i18n.Text("Scroll / navigate"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("l"), dim.Render(i18n.Text("  Choose display language"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("q  Ctrl+C"), dim.Render(i18n.Text("Quit"))))
	lines = append(lines, "")
	lines = append(lines, bright.Render(i18n.Text("  Lesson Phases")))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("→  / Enter"), dim.Render(i18n.Text("Advance to next phase"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("←  / Bksp"), dim.Render(i18n.Text("Go back to previous phase"))))
	lines = append(lines, "")
	lines = append(lines, bright.Render(i18n.Text("  Visualizations")))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("Enter / Space"), dim.Render(i18n.Text("Interact with visualization"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("r"), dim.Render(i18n.Text("Reset visualization"))))
	lines = append(lines, "")
	lines = append(lines, bright.Render(i18n.Text("  Quiz")))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("1-4"), dim.Render(i18n.Text("Select answer (multiple choice)"))))
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render("Enter"), dim.Render(i18n.Text("Submit answer"))))
	lines = append(lines, "")
	lines = append(lines, bright.Render(i18n.Text("  Each lesson follows: Theory -> Visualization -> Quiz")))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Press any key to close")))

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

func (m AppModel) viewCourseComplete() string {
	accent := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
	bright := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true)
	dim := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	green := lipgloss.NewStyle().Foreground(ui.ColorBeginner).Bold(true)
	link := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Underline(true)

	var lines []string
	lines = append(lines, "")
	lines = append(lines, green.Render(i18n.Text("  Congratulations!")))
	lines = append(lines, "")
	lines = append(lines, bright.Render(i18n.Textf("  You've completed all %d lessons.", len(m.lessons))))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  You now understand the core concepts behind")))
	lines = append(lines, dim.Render(i18n.Text("  AI-assisted development: context windows, tools,")))
	lines = append(lines, dim.Render(i18n.Text("  MCP, subagents, batch execution, and more.")))
	lines = append(lines, "")
	lines = append(lines, accent.Render(i18n.Text("  ── What's Next? ──")))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Put these concepts into practice! Try using an")))
	lines = append(lines, dim.Render(i18n.Text("  AI coding assistant with your own projects and")))
	lines = append(lines, dim.Render(i18n.Text("  see how these patterns apply in real workflows.")))
	lines = append(lines, "")
	lines = append(lines, accent.Render(i18n.Text("  ── Contribute ──")))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Something missing? Something wrong? We'd love your help.")))
	lines = append(lines, dim.Render(i18n.Text("  Open an issue or submit a PR:")))
	lines = append(lines, "")
	lines = append(lines, "  "+link.Render("github.com/naorpeled/aitutor"))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Whether it's a new lesson idea, a bug fix, or")))
	lines = append(lines, dim.Render(i18n.Text("  better explanations — all contributions welcome.")))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Press p to revisit lessons  |  q to quit")))

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBeginner).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

func (m AppModel) viewLanguageSelector() string {
	accent := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
	dim := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	highlight := lipgloss.NewStyle().Foreground(ui.ColorHighlight).Bold(true)

	var lines []string
	lines = append(lines, accent.Render(i18n.Text("  Language")))
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  Choose display language")))
	lines = append(lines, "")
	for idx, lang := range i18n.Languages() {
		prefix := "  "
		style := dim
		if idx == m.languageCursor {
			prefix = "▸ "
			style = highlight
		}
		lines = append(lines, prefix+style.Render(i18n.LanguageName(lang)))
	}
	lines = append(lines, "")
	lines = append(lines, dim.Render(i18n.Text("  [↑/↓] Navigate  [Enter] Apply  [Esc] Close")))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m AppModel) updateLanguageSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.showLanguage = false
		return m, nil
	case "up", "k":
		if m.languageCursor > 0 {
			m.languageCursor--
		}
		return m, nil
	case "down", "j":
		if m.languageCursor < len(i18n.Languages())-1 {
			m.languageCursor++
		}
		return m, nil
	case "enter":
		m.applyLanguage(i18n.Languages()[m.languageCursor])
		m.showLanguage = false
		return m, nil
	}
	return m, nil
}
