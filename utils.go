package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func logMessage(message string) error {
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().UnixMilli()
	_, err = fmt.Fprintf(file, "[%d] %s\n", timestamp, message)
	return err
}

func printHelp() {
	helpText := lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#FF00FF")).
	Padding(1, 2).
	Render(
		fmt.Sprintf(
			"%s\nUsage:\n"+
			"  %szcr install <package>%s - Install a package\n"+
			"  %szcr find <package>%s - Search for a package\n"+
			"  %szcr remove <package>%s - Remove a package\n"+
			"  %szcr update <package>%s - Update a specific package\n"+
			"  %szcr update-all%s - Update all installed packages\n"+
			"  %szcr autoremove%s - Remove temporary files and logs\n"+
			"  %szcr help%s - Show this help message\n"+
			"  %szcr how-to-add%s - Instructions for adding new repositories\n",
	      titleStyle.Render(" zcr - Zenit Linux Package Manager "),
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
			    successStyle.Render(""), "",
		),
	)
	fmt.Println(helpText)
}

func printHowToAdd() {
	howToAddText := lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#FF00FF")).
	Padding(1, 2).
	Render(
		fmt.Sprintf(
			"%s\n"+
			"You can contribute your own repository to zcr by submitting it to:\n"+
			" - %sIssues: %s%s\n"+
			" - %sDiscussions: %s%s\n"+
			"Please read the documentation for more details:\n"+
			" - %shttps://github.com/Zenit-Linux/zcr/blob/main/README.md%s\n",
	      titleStyle.Render(" How to Add a Repository to zcr "),
			    infoStyle.Render(""), infoStyle.Render("https://github.com/Zenit-Linux/zcr/issues"), "",
			    infoStyle.Render(""), infoStyle.Render("https://github.com/Zenit-Linux/zcr/discussions"), "",
			    infoStyle.Render(""), "",
		),
	)
	fmt.Println(howToAddText)
}
