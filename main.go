package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const logFilePath = "/tmp/zcr.logs"

// Styles for UI
var (
	// Gold as the primary color
	goldStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFD700")).
	Bold(true)

	// Title style with a border
	titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFD700")).
	Background(lipgloss.Color("#1A1A1A")).
	Bold(true).
	Padding(0, 1).
	MarginBottom(1).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#FF00FF")) // Magenta border

	// Subtitle for progress
	subtitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF69B4")). // Hot pink
	Bold(true).
	MarginBottom(1)

	// Error messages in red
	errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF0000")).
	Bold(true)

	// Success messages in gold
	successStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFD700")).
	Bold(true)

	// Info messages in cyan
	infoStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#00FFFF"))

	// Progress stages in orange
	progressStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFA500")).
	Bold(true)

	// Highlight for selected items
	highlightStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#800080")). // Purple
	Bold(true)

	// Key bindings
	keyEnter = key.NewBinding(key.WithKeys("enter"))
	keyUp    = key.NewBinding(key.WithKeys("up"))
	keyDown  = key.NewBinding(key.WithKeys("down"))
	keyQuit  = key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"))
)

func main() {
	if err := logMessage("Starting zcr execution"); err != nil {
		fmt.Printf("%sError logging: %v%s\n", errorStyle.Render("✖ "), err, "")
		os.Exit(1)
	}

	args := os.Args
	if len(args) < 2 {
		printHelp()
		logMessage("No command provided, showing help")
		os.Exit(0)
	}

	command := args[1]
	logMessage(fmt.Sprintf("Received command: %s", command))

	switch command {
		case "install":
			if len(args) != 3 {
				fmt.Printf("%sUsage: zcr install <package>%s\n", errorStyle.Render("✖ "), "")
				logMessage(fmt.Sprintf("Invalid install command: expected 3 arguments, got %d", len(args)))
				os.Exit(1)
			}
			installPackage(args[2])
		case "find":
			if len(args) != 3 {
				fmt.Printf("%sUsage: zcr find <package>%s\n", errorStyle.Render("✖ "), "")
				logMessage(fmt.Sprintf("Invalid find command: expected 3 arguments, got %d", len(args)))
				os.Exit(1)
			}
			findPackage(args[2])
		case "remove":
			if len(args) != 3 {
				fmt.Printf("%sUsage: zcr remove <package>%s\n", errorStyle.Render("✖ "), "")
				logMessage(fmt.Sprintf("Invalid remove command: expected 3 arguments, got %d", len(args)))
				os.Exit(1)
			}
			removePackage(args[2])
		case "update":
			if len(args) != 3 {
				fmt.Printf("%sUsage: zcr update <package>%s\n", errorStyle.Render("✖ "), "")
				logMessage(fmt.Sprintf("Invalid update command: expected 3 arguments, got %d", len(args)))
				os.Exit(1)
			}
			updatePackage(args[2])
		case "update-all":
			updateAllPackages()
		case "autoremove":
			autoremove()
		case "?", "help":
			printHelp()
			logMessage("Displayed help message")
		case "how-to-add":
			printHowToAdd()
			logMessage("Displayed how-to-add instructions")
		default:
			fmt.Printf("%sUnknown command: %s%s\n", errorStyle.Render("✖ "), command, "")
			logMessage(fmt.Sprintf("Unknown command: %s", command))
			printHelp()
			os.Exit(1)
	}
}

// ConfirmationModel for Bubble Tea GUI
type ConfirmationModel struct {
	choices     []string
	cursor      int
	confirmed   bool
	packageName string
	quitting    bool
	err         error
}

func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
				case key.Matches(msg, keyUp):
					if m.cursor > 0 {
						m.cursor--
					}
				case key.Matches(msg, keyDown):
					if m.cursor < len(m.choices)-1 {
						m.cursor++
					}
				case key.Matches(msg, keyEnter):
					m.confirmed = m.cursor == 0 // 0 is "Yes"
					m.quitting = true
					return m, tea.Quit
				case key.Matches(msg, keyQuit):
					m.quitting = true
					m.confirmed = false
					return m, tea.Quit
			}
	}
	return m, nil
}

func (m ConfirmationModel) View() string {
	// Table-like structure for confirmation
	s := titleStyle.Render(fmt.Sprintf(" Confirm Installation of %s ", m.packageName)) + "\n"
	table := lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#FF00FF")). // Magenta border
	Padding(0, 2).
	Width(40)

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = highlightStyle.Render("➤ ")
		}
		s += table.Render(fmt.Sprintf("%s%s", cursor, choice)) + "\n"
	}
	s += infoStyle.Render("\nUse ↑↓ to select, Enter to confirm, q to quit.")
	return s
}

func confirmInstall(packageName string) bool {
	model := ConfirmationModel{
		choices:     []string{"Yes", "No"},
		packageName: packageName,
	}
	p := tea.NewProgram(model)
	m, err := p.Run()
	if err != nil {
		fmt.Printf("%sError running confirmation: %v%s\n", errorStyle.Render("✖ "), err, "")
		return false
	}
	confModel := m.(ConfirmationModel)
	return confModel.confirmed && !confModel.quitting
}

// ProgressModel for installation progress
type ProgressModel struct {
	spinner     spinner.Model
	stages      []string
	current     int
	packageName string
	quitting    bool
	err         error
	done        bool
}

func NewProgressModel(packageName string) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = progressStyle
	return ProgressModel{
		spinner:     s,
		stages:      []string{"Fetching repo list", "Searching for package", "Cloning repository", "Executing unpack.sh", "Installation complete"},
		packageName: packageName,
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.nextStage())
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, keyQuit) {
				m.quitting = true
				return m, tea.Quit
			}
		case errMsg:
			m.err = msg
			m.done = true
			return m, tea.Quit
		case progressMsg:
			m.current++
			if m.current >= len(m.stages) {
				m.done = true
				return m, tea.Quit
			}
			return m, m.nextStage()
		default:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
	}
	return m, nil
}

func (m ProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("✖ Error: %v", m.err))
		}
		return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF00FF")).
		Padding(0, 2).
		Render(successStyle.Render(fmt.Sprintf("✔ Package %s installed successfully!", m.packageName)))
	}
	if m.quitting {
		return errorStyle.Render("✖ Installation cancelled.")
	}
	// Progress bar simulation
	progressBar := lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF00FF")).
	Render("[" + strings.Repeat("█", m.current*2) + strings.Repeat(" ", (len(m.stages)-m.current)*2) + "]")
	return lipgloss.JoinVertical(
		lipgloss.Left,
		subtitleStyle.Render(fmt.Sprintf(" Installing %s ", m.packageName)),
				     fmt.Sprintf("%s %s", m.spinner.View(), progressStyle.Render(m.stages[m.current])),
				     progressBar,
			      infoStyle.Render("\nPress q to cancel."),
	)
}

type progressMsg struct{}
type errMsg error

func (m ProgressModel) nextStage() tea.Cmd {
	return func() tea.Msg {
		return progressMsg{}
	}
}
