package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Set up logging
	logFile, err := os.OpenFile("/tmp/zcr.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Parse command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: zcr <command> [arguments]")
		fmt.Println("Commands: ui, install, remove, update, upgrade, find, refresh")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
		case "ui":
			// Launch the TUI
			p := tea.NewProgram(initialModel(), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				log.Printf("Error running TUI: %v", err)
				fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
				os.Exit(1)
			}
		case "install":
			pkg := flag.String("pkg", "", "Package name to install")
			flag.CommandLine.Parse(os.Args[2:])
			if *pkg == "" {
				fmt.Println("Error: package name required for install")
				os.Exit(1)
			}
			m := &model{packages: make(map[string]string)}
			if err := m.loadPackages(); err != nil {
				log.Printf("Error loading packages: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := m.install(*pkg); err != nil {
				log.Printf("Error installing package %s: %v", *pkg, err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Package %s installed successfully.\n", *pkg)
		case "remove":
			pkg := flag.String("pkg", "", "Package name to remove")
			flag.CommandLine.Parse(os.Args[2:])
			if *pkg == "" {
				fmt.Println("Error: package name required for remove")
				os.Exit(1)
			}
			m := &model{packages: make(map[string]string)}
			if err := m.loadPackages(); err != nil {
				log.Printf("Error loading packages: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := m.remove(*pkg); err != nil {
				log.Printf("Error removing package %s: %v", *pkg, err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Package %s removed successfully.\n", *pkg)
		case "update":
			pkg := flag.String("pkg", "", "Package name to update")
			flag.CommandLine.Parse(os.Args[2:])
			if *pkg == "" {
				fmt.Println("Error: package name required for update")
				os.Exit(1)
			}
			m := &model{packages: make(map[string]string)}
			if err := m.loadPackages(); err != nil {
				log.Printf("Error loading packages: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := m.update(*pkg); err != nil {
				log.Printf("Error updating package %s: %v", *pkg, err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Package %s updated successfully.\n", *pkg)
		case "upgrade":
			m := &model{packages: make(map[string]string)}
			if err := m.loadPackages(); err != nil {
				log.Printf("Error loading packages: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := m.upgrade(); err != nil {
				log.Printf("Error upgrading packages: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("All packages upgraded successfully.")
		case "find":
			query := flag.String("query", "", "Search query for packages")
			flag.CommandLine.Parse(os.Args[2:])
			if *query == "" {
				fmt.Println("Error: search query required for find")
				os.Exit(1)
			}
			m := &model{packages: make(map[string]string), query: *query}
			if err := m.loadPackages(); err != nil {
				log.Printf("Error loading packages: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			// Call find and assert the tea.Model to *model
			updatedModel, _ := m.find()
			m, ok := updatedModel.(*model)
			if !ok {
				log.Println("Error: type assertion failed for find result")
				fmt.Fprintln(os.Stderr, "Error: internal type assertion failed")
				os.Exit(1)
			}
			if m.state == stateResult {
				fmt.Println(m.result)
			} else {
				for _, i := range m.list.Items() {
					pkg, ok := i.(item)
					if !ok {
						log.Println("Error: type assertion failed for list item")
						continue
					}
					fmt.Printf("%s: %s\n", pkg.title, pkg.desc)
				}
			}
		case "refresh":
			m := &model{packages: make(map[string]string)}
			if err := m.loadPackages(); err != nil {
				log.Printf("Error refreshing package list: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Package list refreshed successfully.")
		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Commands: ui, install, remove, update, upgrade, find, refresh")
			os.Exit(1)
	}
}
