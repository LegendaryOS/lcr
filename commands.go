package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
)

func (m *model) loadPackages() error {
	log.Println("Loading packages...")
	path, err := downloadRepoList()
	if err != nil {
		return err
	}
	m.packages, err = parseRepoList(path)
	if err != nil {
		return err
	}
	log.Println("Packages loaded successfully.")
	return nil
}

func downloadRepoList() (string, error) {
	log.Println("Downloading repo list...")
	url := "https://raw.githubusercontent.com/Zenit-Linux/zcr/main/library/repo-list.zcr"
	path := "/tmp/repo-list.zcr"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}
	log.Println("Repo list downloaded.")
	return path, nil
}

func parseRepoList(path string) (map[string]string, error) {
	log.Println("Parsing repo list...")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	packages := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " -> ", 2)
		if len(parts) == 2 {
			packages[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	log.Println("Repo list parsed.")
	return packages, nil
}

func (m *model) install(pakiet string) error {
	log.Printf("Installing package: %s\n", pakiet)
	url, ok := m.packages[pakiet]
	if !ok {
		err := fmt.Errorf("package %s not found", pakiet)
		log.Println(err)
		return err
	}
	dest := filepath.Join("/usr/lib/zcr", pakiet)
	_, err := git.PlainClone(dest, false, &git.CloneOptions{URL: url})
	if err != nil {
		log.Println("Clone error:", err)
		return err
	}
	err = m.runUnpack(dest)
	if err != nil {
		log.Println("Unpack error:", err)
		return err
	}
	log.Printf("Package %s installed.\n", pakiet)
	return nil
}

func (m *model) runUnpack(dest string) error {
	log.Println("Running unpack.sh...")
	buildDir := filepath.Join(dest, "zcr-build-files")
	unpack := filepath.Join(buildDir, "unpack.sh")
	err := os.Chmod(unpack, 0755)
	if err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", unpack)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println("unpack.sh executed.")
	return nil
}

func (m *model) remove(pakiet string) error {
	log.Printf("Removing package: %s\n", pakiet)
	dest := filepath.Join("/usr/lib/zcr", pakiet)
	buildDir := filepath.Join(dest, "zcr-build-files")
	removeSh := filepath.Join(buildDir, "remove.sh")
	if _, err := os.Stat(removeSh); err == nil {
		cmd := exec.Command("/bin/sh", removeSh)
		cmd.Dir = buildDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Warning: remove.sh failed for %s: %v\n", pakiet, err)
		}
	}
	err := os.RemoveAll(dest)
	if err != nil {
		log.Println("Remove directory error:", err)
		return err
	}
	log.Printf("Package %s removed.\n", pakiet)
	return nil
}

func (m *model) update(pakiet string) error {
	log.Printf("Updating package: %s\n", pakiet)
	dest := filepath.Join("/usr/lib/zcr", pakiet)
	repo, err := git.PlainOpen(dest)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err == git.NoErrAlreadyUpToDate {
		m.result = successStyle.Render("Already the latest version.")
		log.Println("Already up to date.")
		return nil
	} else if err != nil {
		log.Println("Pull error:", err)
		return err
	}
	err = m.runUnpack(dest)
	if err != nil {
		return err
	}
	log.Printf("Package %s updated.\n", pakiet)
	return nil
}

func (m *model) upgrade() error {
	log.Println("Upgrading all packages...")
	dir := "/usr/lib/zcr"
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			err := m.update(f.Name())
			if err != nil {
				log.Printf("Failed to update %s: %v\n", f.Name(), err)
			}
		}
	}
	log.Println("Upgrade complete.")
	return nil
}

func (m *model) find() (tea.Model, tea.Cmd) {
	log.Printf("Searching for packages with query: %s\n", m.query)
	var items []list.Item
	for pkg, url := range m.packages {
		if strings.Contains(strings.ToLower(pkg), strings.ToLower(m.query)) {
			items = append(items, item{title: pkg, desc: url})
		}
	}
	if len(items) == 0 {
		m.result = infoStyle.Render("No packages found.")
		m.state = stateResult
		log.Println("No packages found.")
		return m, nil
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle.Foreground(greenColor).Bold(true)
	delegate.Styles.NormalTitle.Foreground(goldColor)
	l := list.New(items, delegate, 0, 0)
	l.Title = "Found Packages"
	l.Styles.Title = subtitleStyle
	m.list = l
	log.Println("Search results displayed.")
	return m, nil
}
