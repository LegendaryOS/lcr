package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func fetchRepoList() (string, error) {
	repoURL := "https://raw.githubusercontent.com/Zenit-Linux/zcr/main/library/repo-list.zcr"
	fmt.Printf("%sFetching repository list...%s\n", progressStyle.Render("➜ "), "")
	logMessage(fmt.Sprintf("Fetching repo list from %s", repoURL))

	cmd := exec.Command("curl", "-s", repoURL)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Printf("%sFailed to fetch repo list: %v%s\n", errorStyle.Render("✖ "), err, "")
		logMessage(fmt.Sprintf("Failed to fetch repo list, error: %v", err))
		return "", err
	}

	if err := os.MkdirAll("/tmp", 0755); err != nil {
		logMessage(fmt.Sprintf("Failed to create /tmp directory: %v", err))
		return "", err
	}

	if err := os.WriteFile("/tmp/repo-list.zcr", stdout, 0644); err != nil {
		logMessage(fmt.Sprintf("Failed to save repo list: %v", err))
		return "", err
	}

	logMessage("Saved repo list to /tmp/repo-list.zcr")
	return string(stdout), nil
}

func parseRepoList(content, packageName string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, " -> ")
		if len(parts) != 2 {
			continue
		}
		pkg := strings.TrimSpace(parts[0])
		repo := strings.TrimSpace(parts[1])
		if pkg == packageName {
			logMessage(fmt.Sprintf("Found package %s with repo %s", packageName, repo))
			return repo, nil
		}
	}
	logMessage(fmt.Sprintf("Package %s not found in repo list", packageName))
	return "", fmt.Errorf("package not found")
}

func installPackage(packageName string) {
	if !confirmInstall(packageName) {
		fmt.Printf("%sInstallation of %s cancelled%s\n", errorStyle.Render("✖ "), goldStyle.Render(packageName), "")
		logMessage(fmt.Sprintf("Installation of %s cancelled", packageName))
		return
	}

	p := tea.NewProgram(NewProgressModel(packageName))
	go func() {
		repoList, err := fetchRepoList()
		if err != nil {
			p.Send(errMsg(err))
			return
		}
		p.Send(progressMsg{})

		repoURL, err := parseRepoList(repoList, packageName)
		if err != nil {
			p.Send(errMsg(err))
			return
		}
		p.Send(progressMsg{})

		fmt.Printf("%sInstalling package %s from %s%s\n", successStyle.Render("✔ "), goldStyle.Render(packageName), infoStyle.Render(repoURL), "")
		logMessage(fmt.Sprintf("Cloning package %s from %s", packageName, repoURL))

		installDir := fmt.Sprintf("/usr/lib/zcr/%s", packageName)
		if err := os.MkdirAll(installDir, 0755); err != nil {
			p.Send(errMsg(err))
			return
		}
		logMessage(fmt.Sprintf("Created install directory %s", installDir))

		cmd := exec.Command("git", "clone", repoURL, installDir)
		if err := cmd.Run(); err != nil {
			fmt.Printf("%sFailed to clone repository%s\n", errorStyle.Render("✖ "), "")
			logMessage(fmt.Sprintf("Failed to clone repository for %s, error: %v", packageName, err))
			p.Send(errMsg(err))
			return
		}
		p.Send(progressMsg{})

		logMessage(fmt.Sprintf("Successfully cloned %s to %s", packageName, installDir))

		unpackScript := filepath.Join(installDir, "zcr-build-files", "unpack.sh")
		if _, err := os.Stat(unpackScript); os.IsNotExist(err) {
			fmt.Printf("%sNo unpack.sh found, package cloned to %s%s\n", infoStyle.Render("ℹ "), goldStyle.Render(installDir), "")
			logMessage(fmt.Sprintf("No unpack.sh found for %s, package cloned to %s", packageName, installDir))
			p.Send(progressMsg{})
			return
		}

		fmt.Printf("%sExecuting unpack.sh for %s%s\n", progressStyle.Render("➜ "), goldStyle.Render(packageName), "")
		logMessage(fmt.Sprintf("Executing unpack.sh for %s", packageName))

		if err := exec.Command("sudo", "chmod", "+x", unpackScript).Run(); err != nil {
			fmt.Printf("%sFailed to make unpack.sh executable%s\n", errorStyle.Render("✖ "), "")
			logMessage(fmt.Sprintf("Failed to chmod unpack.sh for %s, error: %v", packageName, err))
			p.Send(errMsg(err))
			return
		}

		if err := exec.Command("sudo", "sh", unpackScript).Run(); err != nil {
			fmt.Printf("%sFailed to execute unpack.sh%s\n", errorStyle.Render("✖ "), "")
			logMessage(fmt.Sprintf("Failed to execute unpack.sh for %s, error: %v", packageName, err))
			p.Send(errMsg(err))
			return
		}
		p.Send(progressMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("%sError during installation: %v%s\n", errorStyle.Render("✖ "), err, "")
	}
}

func findPackage(packageName string) {
	repoList, err := fetchRepoList()
	if err != nil {
		fmt.Printf("%sFailed to fetch repo list: %v%s\n", errorStyle.Render("✖ "), err, "")
		return
	}

	repoURL, err := parseRepoList(repoList, packageName)
	if err != nil {
		fmt.Printf("%sPackage %s not found%s\n", errorStyle.Render("✖ "), goldStyle.Render(packageName), "")
		return
	}

	fmt.Printf("%sFound package %s at %s%s\n", successStyle.Render("✔ "), goldStyle.Render(packageName), infoStyle.Render(repoURL), "")
	logMessage(fmt.Sprintf("Found package %s at %s", packageName, repoURL))
}

func removePackage(packageName string) {
	logMessage(fmt.Sprintf("Starting removal of package %s", packageName))
	installDir := fmt.Sprintf("/usr/lib/zcr/%s", packageName)
	removeScript := filepath.Join(installDir, "zcr-build-files", "remove.sh")

	if _, err := os.Stat(removeScript); err == nil {
		fmt.Printf("%sExecuting remove.sh for %s%s\n", progressStyle.Render("➜ "), goldStyle.Render(packageName), "")
		logMessage(fmt.Sprintf("Executing remove.sh for %s", packageName))
		if err := exec.Command("sudo", "chmod", "+x", removeScript).Run(); err != nil {
			fmt.Printf("%sFailed to make remove.sh executable: %v%s\n", errorStyle.Render("✖ "), err, "")
			logMessage(fmt.Sprintf("Failed to chmod remove.sh for %s, error: %v", packageName, err))
		}
		if err := exec.Command("sudo", "sh", removeScript).Run(); err != nil {
			fmt.Printf("%sFailed to execute remove.sh: %v%s\n", errorStyle.Render("✖ "), err, "")
			logMessage(fmt.Sprintf("Failed to execute remove.sh for %s, error: %v", packageName, err))
		} else {
			logMessage(fmt.Sprintf("Successfully executed remove.sh for %s", packageName))
		}
	} else if os.IsNotExist(err) {
		fmt.Printf("%sNo remove.sh found, proceeding to delete package directory%s\n", infoStyle.Render("ℹ "), "")
		logMessage(fmt.Sprintf("No remove.sh found for %s, proceeding to delete directory", packageName))
	}

	if err := os.RemoveAll(installDir); err != nil {
		fmt.Printf("%sFailed to remove package directory: %v%s\n", errorStyle.Render("✖ "), err, "")
		logMessage(fmt.Sprintf("Failed to remove package %s directory: %v", packageName, err))
		return
	}

	fmt.Printf("%sPackage %s removed successfully%s\n", successStyle.Render("✔ "), goldStyle.Render(packageName), "")
	logMessage(fmt.Sprintf("Package %s removed successfully", packageName))
}

func updatePackage(packageName string) {
	logMessage(fmt.Sprintf("Starting update of package %s", packageName))
	installDir := fmt.Sprintf("/usr/lib/zcr/%s", packageName)

	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		fmt.Printf("%sPackage %s is not installed%s\n", errorStyle.Render("✖ "), goldStyle.Render(packageName), "")
		logMessage(fmt.Sprintf("Package %s is not installed, cannot update", packageName))
		return
	}

	// Check if update is needed
	cmd := exec.Command("git", "-C", installDir, "fetch", "origin")
	if err := cmd.Run(); err != nil {
		fmt.Printf("%sFailed to fetch updates for %s: %v%s\n", errorStyle.Render("✖ "), goldStyle.Render(packageName), err, "")
		logMessage(fmt.Sprintf("Failed to fetch updates for %s: %v", packageName, err))
		// Proceed with update on error
	} else {
		cmd = exec.Command("git", "-C", installDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("%sNo tracked branch for %s, assuming update needed%s\n", infoStyle.Render("ℹ "), goldStyle.Render(packageName), "")
			logMessage(fmt.Sprintf("No tracked branch for %s: %v", packageName, err))
		} else {
			remoteBranch := strings.TrimSpace(string(output))
			cmd = exec.Command("git", "-C", installDir, "rev-list", "--count", "HEAD.." + remoteBranch)
			countOut, err := cmd.Output()
			if err != nil {
				fmt.Printf("%sFailed to check update count for %s: %v%s\n", errorStyle.Render("✖ "), goldStyle.Render(packageName), err, "")
				logMessage(fmt.Sprintf("Failed to check update count for %s: %v", packageName, err))
			} else {
				countStr := strings.TrimSpace(string(countOut))
				count, err := strconv.Atoi(countStr)
				if err != nil {
					fmt.Printf("%sInvalid update count for %s: %s%s\n", errorStyle.Render("✖ "), goldStyle.Render(packageName), countStr, "")
					logMessage(fmt.Sprintf("Invalid update count for %s: %v", packageName, err))
				} else if count == 0 {
					fmt.Printf("%sPackage %s is up to date%s\n", infoStyle.Render("ℹ "), goldStyle.Render(packageName), "")
					logMessage(fmt.Sprintf("Package %s is up to date", packageName))
					return
				}
			}
		}
	}

	fmt.Printf("%sUpdate available for %s. Updating...%s\n", progressStyle.Render("➜ "), goldStyle.Render(packageName), "")
	logMessage(fmt.Sprintf("Update available for %s", packageName))

	// Run remove.sh if exists
	removeScript := filepath.Join(installDir, "zcr-build-files", "remove.sh")
	if _, err := os.Stat(removeScript); err == nil {
		fmt.Printf("%sExecuting remove.sh for update of %s%s\n", progressStyle.Render("➜ "), goldStyle.Render(packageName), "")
		logMessage(fmt.Sprintf("Executing remove.sh for update of %s", packageName))
		if err := exec.Command("sudo", "chmod", "+x", removeScript).Run(); err != nil {
			fmt.Printf("%sFailed to make remove.sh executable: %v%s\n", errorStyle.Render("✖ "), err, "")
			logMessage(fmt.Sprintf("Failed to chmod remove.sh for %s: %v", packageName, err))
		}
		if err := exec.Command("sudo", "sh", removeScript).Run(); err != nil {
			fmt.Printf("%sFailed to execute remove.sh: %v%s\n", errorStyle.Render("✖ "), err, "")
			logMessage(fmt.Sprintf("Failed to execute remove.sh for %s: %v", packageName, err))
		} else {
			logMessage(fmt.Sprintf("Successfully executed remove.sh for update of %s", packageName))
		}
	}

	// Delete the directory
	if err := os.RemoveAll(installDir); err != nil {
		fmt.Printf("%sFailed to remove old package directory: %v%s\n", errorStyle.Render("✖ "), err, "")
		logMessage(fmt.Sprintf("Failed to remove old package %s directory: %v", packageName, err))
		return
	}

	// Re-install (clone and unpack)
	installPackage(packageName)
}

func updateAllPackages() {
	logMessage("Starting update-all operation")
	repoList, err := fetchRepoList()
	if err != nil {
		fmt.Printf("%sFailed to fetch repo list: %v%s\n", errorStyle.Render("✖ "), err, "")
		return
	}

	fmt.Printf("%sUpdating all packages...%s\n", progressStyle.Render("➜ "), "")
	scanner := bufio.NewScanner(strings.NewReader(repoList))
	updatedCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, " -> ")
		if len(parts) != 2 {
			continue
		}
		pkg := strings.TrimSpace(parts[0])
		updatePackage(pkg)
		updatedCount++ // Count only if updated, but for simplicity, or add logic
	}

	fmt.Printf("%sAll packages updated (%d processed)%s\n", successStyle.Render("✔ "), updatedCount, "")
	logMessage("All packages updated successfully")
}

func autoremove() {
	fmt.Printf("%sCleaning up temporary files...%s\n", progressStyle.Render("➜ "), "")
	logMessage("Starting autoremove operation")

	tempFiles := []string{"/tmp/repo-list.zcr"}
	for _, file := range tempFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			logMessage(fmt.Sprintf("Temporary file %s not found, skipping", file))
			continue
		}
		if err := os.Remove(file); err != nil {
			fmt.Printf("%sFailed to remove %s: %v%s\n", errorStyle.Render("✖ "), file, err, "")
			logMessage(fmt.Sprintf("Failed to remove temporary file %s: %v", file, err))
			continue
		}
		fmt.Printf("%sRemoved %s%s\n", successStyle.Render("✔ "), infoStyle.Render(file), "")
		logMessage(fmt.Sprintf("Removed temporary file %s", file))
	}

	fmt.Printf("%sCleanup completed%s\n", successStyle.Render("✔ "), "")
	logMessage("Autoremove completed")
}
