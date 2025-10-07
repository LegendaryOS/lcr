package main

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	goldColor   = lipgloss.Color("#FFD700")
	greenColor  = lipgloss.Color("#00FF00")
	redColor    = lipgloss.Color("#FF0000")
	blueColor   = lipgloss.Color("#0000FF")
	purpleColor = lipgloss.Color("#800080")
	yellowColor = lipgloss.Color("#FFFF00")

	titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(goldColor).
	Background(lipgloss.Color("#000000")).
	Padding(1, 2).
	Margin(1, 0).
	Border(lipgloss.RoundedBorder(), true).
	BorderForeground(yellowColor).
	Align(lipgloss.Center).
	Width(60)

	subtitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(blueColor).
	Padding(0, 1).
	Margin(1, 0).
	Border(lipgloss.DoubleBorder(), true).
	BorderForeground(blueColor)

	successStyle = lipgloss.NewStyle().
	Foreground(greenColor).
	Bold(true).
	Padding(1).
	Border(lipgloss.NormalBorder(), true).
	BorderForeground(greenColor)

	errorStyle = lipgloss.NewStyle().
	Foreground(redColor).
	Bold(true).
	Padding(1).
	Border(lipgloss.NormalBorder(), true).
	BorderForeground(redColor)

	infoStyle = lipgloss.NewStyle().
	Foreground(purpleColor).
	Italic(true).
	Padding(0, 1)

	listStyle = lipgloss.NewStyle().
	Margin(1, 2).
	Border(lipgloss.ThickBorder(), true).
	BorderForeground(goldColor)

	docStyle = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(yellowColor).Align(lipgloss.Center).Margin(1)
	footerStyle = lipgloss.NewStyle().Foreground(purpleColor).Align(lipgloss.Center).Margin(1)
)

type state string

const (
	stateMenu       state = "menu"
	stateInputPakiet state = "input_pakiet"
	stateFindQuery  state = "find_query"
	stateExec       state = "exec"
	stateResult     state = "result"
	stateHelp       state = "help"
	stateHowToAdd   state = "how_to_add"
	stateList       state = "list"
)

type model struct {
	state     state
	choice    string
	pakiet    string
	query     string
	result    string
	list      list.Model
	textinput textinput.Model
	packages  map[string]string
	err       error
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func initialModel() *model {
	ti := textinput.New()
	ti.Placeholder = "Enter package name..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30
	ti.TextStyle = lipgloss.NewStyle().Foreground(goldColor)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(blueColor)

	items := []list.Item{
		item{title: "install", desc: "Install a package"},
		item{title: "remove", desc: "Remove a package"},
		item{title: "update", desc: "Update a package"},
		item{title: "upgrade", desc: "Upgrade all packages"},
		item{title: "find", desc: "Find packages"},
		item{title: "refresh", desc: "Refresh package list"},
		item{title: "help", desc: "Show help"},
		item{title: "how-to-add", desc: "How to add your own repo"},
		item{title: "exit", desc: "Exit the application"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle.Foreground(goldColor).Bold(true)
	delegate.Styles.NormalTitle.Foreground(blueColor)
	delegate.Styles.SelectedDesc.Foreground(greenColor)
	delegate.Styles.NormalDesc.Foreground(purpleColor)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Zenit Community Repository"
	l.Styles.Title = titleStyle

	return &model{
		state:     stateMenu,
		textinput: ti,
		list:      l,
		packages:  make(map[string]string),
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
		case stateMenu:
			switch msg := msg.(type) {
				case tea.KeyMsg:
					if msg.String() == "ctrl+c" || msg.String() == "q" {
						log.Println("User exited via ctrl+c or q")
						return m, tea.Quit
					}
				case tea.WindowSizeMsg:
					h, v := docStyle.GetFrameSize()
					m.list.SetSize(msg.Width-h, msg.Height-v)
			}

			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
				selected := m.list.SelectedItem().(item)
				m.choice = selected.title
				log.Printf("Selected command: %s", m.choice)
				if m.choice == "exit" {
					log.Println("Exiting application")
					return m, tea.Quit
				} else if m.choice == "upgrade" || m.choice == "refresh" || m.choice == "help" || m.choice == "how-to-add" {
					m.state = stateExec
				} else if m.choice == "🔍 find" {
					m.state = stateFindQuery
					m.textinput.Placeholder = "Enter search query..."
					m.textinput.Focus()
					log.Println("Switched to find query state")
				} else {
					m.state = stateInputPakiet
					m.textinput.Placeholder = "Enter package name..."
					m.textinput.Focus()
					log.Println("Switched to input package state")
				}
			}
			return m, cmd

				case stateInputPakiet, stateFindQuery:
					switch msg := msg.(type) {
						case tea.KeyMsg:
							if msg.String() == "esc" {
								log.Println("Cancelled input, returning to menu")
								m.state = stateMenu
								m.textinput.Reset()
								return m, nil
							}
							if msg.String() == "enter" {
								if m.state == stateInputPakiet {
									m.pakiet = m.textinput.Value()
									log.Printf("Package name entered: %s", m.pakiet)
								} else {
									m.query = m.textinput.Value()
									log.Printf("Search query entered: %s", m.query)
								}
								m.textinput.Reset()
								m.state = stateExec
								return m, nil
							}
					}
					var cmd tea.Cmd
					m.textinput, cmd = m.textinput.Update(msg)
					return m, cmd

						case stateExec:
							m.err = m.loadPackages()
							if m.err != nil {
								log.Println("Error loading packages:", m.err)
								m.result = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
								m.state = stateResult
								return m, nil
							}

							switch m.choice {
								case "install":
									m.err = m.install(m.pakiet)
								case "remove":
									m.err = m.remove(m.pakiet)
								case "update":
									m.err = m.update(m.pakiet)
								case "upgrade":
									m.err = m.upgrade()
								case "find":
									m.state = stateList
									return m.find()
								case "refresh":
									m.result = successStyle.Render("Package list refreshed successfully.")
									m.state = stateResult
									log.Println("Package list refresh executed")
									return m, nil
								case "help":
									m.state = stateHelp
									log.Println("Switched to help state")
									return m, nil
								case "how-to-add":
									m.state = stateHowToAdd
									log.Println("Switched to how-to-add state")
									return m, nil
							}

							if m.err != nil {
								log.Println("Execution error:", m.err)
								m.result = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
							} else {
								m.result = successStyle.Render(fmt.Sprintf("%s executed successfully for %s.", m.choice, m.pakiet))
							}
							m.state = stateResult
							return m, nil

								case stateList:
									switch msg := msg.(type) {
										case tea.KeyMsg:
											if msg.String() == "esc" || msg.String() == "q" {
												log.Println("Exiting list view, returning to menu")
												m.state = stateMenu
												return m, nil
											}
										case tea.WindowSizeMsg:
											h, v := docStyle.GetFrameSize()
											m.list.SetSize(msg.Width-h, msg.Height-v)
									}
									var cmd tea.Cmd
									m.list, cmd = m.list.Update(msg)
									return m, cmd

										case stateResult, stateHelp, stateHowToAdd:
											switch msg := msg.(type) {
												case tea.KeyMsg:
													if msg.String() == "enter" || msg.String() == "esc" || msg.String() == "q" {
														log.Println("Returning to menu from", m.state)
														m.state = stateMenu
														m.result = ""
														return m, nil
													}
											}
											return m, nil
	}

	return m, nil
}

func (m *model) View() string {
	header := headerStyle.Render("ZCR - Zenit Community Repository")
	footer := footerStyle.Render("Press q to quit | esc to back")

	switch m.state {
		case stateMenu:
			return docStyle.Render(header + "\n" + m.list.View() + "\n" + footer)
		case stateInputPakiet, stateFindQuery:
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n\n%s to cancel.",
		      header,
		      subtitleStyle.Render("Enter value:"),
					   m.textinput.View(),
					   infoStyle.Render("esc"),
			)
		case stateList:
			return docStyle.Render(header + "\n" + m.list.View() + "\n" + footer)
		case stateResult:
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n\n%s\n\n%s",
		      header,
		      titleStyle.Render("Result"),
					   m.result,
		      infoStyle.Render("Press enter to return to menu."),
					   footer,
			)
		case stateHelp:
			helpText := infoStyle.Render(`Commands:
			- install: Installs the package by cloning its repo and running unpack.sh.
			- remove: Removes the package by running remove.sh and deleting the directory.
			- update: Updates the package to the latest version.
			- upgrade: Updates all installed packages.
			- find: Searches for packages in the repository list.
			- refresh: Refreshes the package list.
			- help: Shows this help.
			- how-to-add: Shows how to add your own repository.
			- exit: Exits the application.`)
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n\n%s\n\n%s",
		      header,
		      titleStyle.Render("Help"),
					   helpText,
		      infoStyle.Render("Press enter to return."),
					   footer,
			)
		case stateHowToAdd:
			howToText := infoStyle.Render(`How to add your own repo:
			- Example repo: https://github.com/Zenit-Linux/Sample-repo-zcr/
			- Guide to creating your own repo: https://github.com/Zenit-Linux/zcr/wiki/Creating-your-own-repository-for-zcr
			- Submit your repo: https://github.com/Zenit-Linux/zcr/discussions or https://github.com/Zenit-Linux/zcr/issues or https://sourceforge.net/p/zenit-linux/discussion/`)
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n\n%s\n\n%s",
		      header,
		      titleStyle.Render("How to Add Repo"),
					   howToText,
		      infoStyle.Render("Press enter to return."),
					   footer,
			)
	}
	return ""
}
