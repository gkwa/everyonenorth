package core

import (
	"bufio"
	"embed"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/gkwa/everyonenorth/util"
)

//go:embed author_template.tmpl
var templateFS embed.FS

type Author struct {
	Name        string
	CommitCount string
	RepoName    string
	SearchURL   string
}

func Run(cwd string) {
	repoName, err := getRepoName(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting repository name:", err)
		return
	}

	currentBranch, err := getCurrentBranch(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current branch:", err)
		return
	}

	output, err := executeGitShortlog(cwd, currentBranch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error executing git shortlog:", err)
		return
	}

	authors, err := parseLogOutput(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing log output:", err)
		return
	}

	authorsWithURL := generateSearchURLs(authors, repoName)

	err = writeMarkdownFile("authors.md", authorsWithURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing markdown file:", err)
		return
	}

	fmt.Println("Markdown file generated successfully.")
}

func executeGitShortlog(cwd, currentBranch string) (string, error) {
	cmd := exec.Command("git", "-C", cwd, "-c", "core.excludesFile=", "shortlog", "--summary", "--numbered", currentBranch)
	output, exitCode, err := util.RunCommand(cmd, cwd)
	if err != nil {
		return "", fmt.Errorf("error executing git shortlog: %v\nExit code: %d\nOutput: %s", err, exitCode, output)
	}
	return output, nil
}

func parseLogOutput(output string) ([]Author, error) {
	var authors []Author
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			authorName := strings.Join(fields[1:], " ")
			author := Author{
				Name:        authorName,
				CommitCount: fields[0],
			}
			authors = append(authors, author)
		}
	}
	return authors, nil
}

func generateSearchURLs(authors []Author, repoName string) []Author {
	var authorsWithURL []Author
	for _, author := range authors {
		searchQuery := fmt.Sprintf("%s %s", author.Name, repoName)
		baseURL := "https://www.google.com/search"
		queryParams := url.Values{
			"tbm": []string{"isch"},
			"q":   []string{searchQuery},
		}
		searchURL := baseURL + "?" + queryParams.Encode()
		author.RepoName = repoName
		author.SearchURL = searchURL
		authorsWithURL = append(authorsWithURL, author)
	}
	return authorsWithURL
}

func writeMarkdownFile(filename string, authors []Author) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	tmpl, err := template.ParseFS(templateFS, "author_template.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err)
	}

	for _, author := range authors {
		err := tmpl.Execute(writer, author)
		if err != nil {
			return fmt.Errorf("error executing template: %v", err)
		}
		_, err = writer.WriteString("\n")
		if err != nil {
			return fmt.Errorf("can't write newline to file: %w", err)
		}
	}

	writer.Flush()
	return nil
}

func getRepoName(cwd string) (string, error) {
	repoURL, err := getRepoURL(cwd)
	if err != nil {
		return "", err
	}

	if isSSHURL(repoURL) {
		return getRepoNameFromSSHURL(repoURL)
	}

	return getRepoNameFromHTTPSURL(repoURL)
}

func getRepoURL(cwd string) (string, error) {
	cmd := exec.Command(
		"git",
		"-C", cwd,
		"config", "--get", "remote.origin.url",
	)
	output, exitCode, err := util.RunCommand(cmd, cwd)
	if err != nil {
		return "", fmt.Errorf("failed to get repository name: %v\nExit code: %d\nOutput: %s", err, exitCode, output)
	}

	return strings.TrimSpace(output), nil
}

func isSSHURL(repoURL string) bool {
	return strings.HasPrefix(repoURL, "git@")
}

func getRepoNameFromSSHURL(repoURL string) (string, error) {
	parts := strings.Split(repoURL, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid SSH URL format: %s", repoURL)
	}

	repoPath := parts[1]
	return extractRepoName(repoPath), nil
}

func getRepoNameFromHTTPSURL(repoURL string) (string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %v", err)
	}

	repoPath := parsedURL.Path
	return extractRepoName(repoPath), nil
}

func extractRepoName(repoPath string) string {
	repoNameParts := strings.Split(repoPath, "/")
	repoName := repoNameParts[len(repoNameParts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")
	return repoName
}

func getCurrentBranch(gitDir string) (string, error) {
	cmd := exec.Command(
		"git",
		"-C", gitDir,
		"rev-parse", "--abbrev-ref", "HEAD",
	)
	output, exitCode, err := util.RunCommand(cmd, gitDir)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %v\nExit code: %d\nOutput: %s", err, exitCode, output)
	}

	currentBranch := strings.TrimSpace(output)
	return currentBranch, nil
}
