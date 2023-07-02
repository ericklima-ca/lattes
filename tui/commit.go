package tui

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/ericklima-ca/lattes/api"
)

type commitModel struct {
	viewport viewport.Model
	textarea textarea.Model
	spin     spinner.Model
	help     help.Model
	editing  bool
	loading  bool
	quitting bool
	accepted bool
	err      error
}

type keyMap struct {
	Space key.Binding
	Esc   key.Binding
	Enter key.Binding
	CtrlC key.Binding
}

var keys = keyMap{
	Esc:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "Cancel")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Accept")),
	Space: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "Edit")),
	CtrlC: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "Cancel")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Esc, k.CtrlC, k.Space, k.Enter}
}
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Esc, k.CtrlC, k.Space, k.Enter},
	}
}

func newCommitModel() *commitModel {
	ta := textarea.New()
	ta.Placeholder = "..."
	ta.Prompt = "┃ "
	ta.CharLimit = 5000
	ta.SetWidth(196)
	ta.SetHeight(6)
	ta.KeyMap.WordForward.SetKeys("ctrl+right")
	ta.KeyMap.WordBackward.SetKeys("ctrl+left")
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = true
	vp := viewport.New(196, 2)
	h := help.New()

	spin := spinner.New()
	spin.Spinner = spinner.Globe

	return &commitModel{
		viewport: vp,
		textarea: ta,
		help:     h,
		spin:     spin,
		loading:  true,
	}
}

type errMsg error

func (m commitModel) Init() tea.Cmd {
	apiCmd := func() tea.Msg {
		patch := m.gitDiff()
		msg, err := api.GetCommitMessage(patch)
		if err != nil {
			return err
		}
		return msg
	}
	return tea.Batch(apiCmd, spinner.Tick)
}

func (m commitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.textarea.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.textarea.Focused() {
				m.editing = false
				m.textarea.Blur()
				readingMessage, _ := glamour.Render("\n# Accept, cancel or edit the commit\n", "dark")
				m.viewport.SetContent(readingMessage)
				viewport.Sync(m.viewport)
				return m, nil
			}
			canceledMessage, _ := glamour.Render("\n# Commit canceled!\n", "dark")
			m.viewport.SetContent(canceledMessage)
			viewport.Sync(m.viewport)
			m.textarea.Reset()
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if !m.textarea.Focused() {
				result := m.textarea.Value()
				acceptedMessage, _ := glamour.Render("\n# Commit accepted!\n", "dark")
				m.viewport.SetContent(acceptedMessage)
				viewport.Sync(m.viewport)
				fmt.Printf("%#v", result)
				m.gitCommit(result)
				output := m.gitLog()
				m.textarea.Reset()
				lines := len(strings.Split(output, "\n"))
				m.textarea.SetHeight(lines)
				m.textarea.SetValue(output)
				m.quitting = true
				m.accepted = true
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		case " ":
			if !m.textarea.Focused() {
				cmdFocus := m.textarea.Focus()
				m.textarea.CursorStart()
				m.editing = true

				editingMessage, _ := glamour.Render("\n# Editing the commit...\n", "dark")
				m.viewport.SetContent(editingMessage)
				viewport.Sync(m.viewport)
				return m, cmdFocus
			} else {
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		default:
			var taCmd tea.Cmd
			m.textarea, taCmd = m.textarea.Update(msg)
			return m, taCmd
		}
	case string:
		m.loading = false
		text, _, _ := m.stylize(msg)
		m.textarea.SetValue(text)
		readingMessage, _ := glamour.Render("\n# Accept, cancel or edit the commit\n", "dark")
		m.viewport.SetContent(readingMessage)
		viewport.Sync(m.viewport)
		var (
			taCmd tea.Cmd
			vpCmd tea.Cmd
		)
		m.textarea, taCmd = m.textarea.Update(msg)
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, tea.Batch(taCmd, vpCmd)
	case errMsg:
		m.loading = false
		errorMsg := lipgloss.
			NewStyle().
			Bold(true).
			Background(lipgloss.Color("#b20000")).
			ColorWhitespace(false).MarginLeft(2).
			SetString("\n API Error: ").
			Render()
		text := msg.Error()
		m.textarea.SetHeight(2)
		m.textarea.SetValue(text)
		m.viewport.SetContent(errorMsg)
		viewport.Sync(m.viewport)
		var (
			taCmd tea.Cmd
			vpCmd tea.Cmd
		)
		m.textarea, taCmd = m.textarea.Update(msg)
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, tea.Batch(tea.Quit, taCmd, vpCmd)
	case spinner.TickMsg:
		if m.loading {
			var spinCmd tea.Cmd
			m.spin, spinCmd = m.spin.Update(msg)
			return m, spinCmd
		}
	}
	return m, nil
}

func (m commitModel) View() string {
	helpText := m.help.View(keys)
	if m.loading {
		msg := fmt.Sprintf("\n# %s Waiting for response...\n", m.spin.View())
		startMessage, _ := glamour.Render(msg, "dark")
		m.viewport.SetContent(startMessage)
		return fmt.Sprintf(m.viewport.View() + "\n" + m.textarea.View() + "\n\n\n\n" + helpText)
	}

	if m.quitting {
		if !m.accepted {
			return fmt.Sprintf(m.viewport.View() + "\n")
		} else {
			return fmt.Sprintf(m.viewport.View() + "\n" + m.textarea.View() + "\n")
		}
	}
	return fmt.Sprintf(m.viewport.View() + "\n" + m.textarea.View() + "\n\n\n\n" + helpText)
}

func (m commitModel) stylize(str string) (string, int, int) {
	text := lipgloss.NewStyle().SetString(str).Render()
	fmt.Printf("%#v", text)
	w, h := lipgloss.Size(text)
	return text, w, h
}

func (m commitModel) gitDiff() string {

	gitDiffCmd := exec.Command("git", "diff", "--staged")

	outGitDiff, err := gitDiffCmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(outGitDiff)
}

func (m commitModel) gitCommit(msg string) {

	gitCommitCmd := exec.Command("git", "commit", "-m", msg)

	_, err := gitCommitCmd.Output()
	if err != nil {
		log.Fatal(err)
	}
}
func (m commitModel) gitLog() string {
	gitLogCmd := exec.Command("git", "log", "-1")

	outGitLog, err := gitLogCmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(outGitLog)
}
func RunCommitProgram() error {
	p := tea.NewProgram(newCommitModel())
	_, err := p.Run()
	return err
}
