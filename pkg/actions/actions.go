package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ActionReference struct {
	Name    string
	Version string
	File    string
}

type OutdatedAction struct {
	File           string `json:"file"`
	Name           string `json:"action"`
	CurrentVersion string `json:"current"`
	LatestVersion  string `json:"latest"`
}

// GitHubTag represents a tag from the GitHub API
type GitHubTag struct {
	Name string `json:"name"`
}

// ErrRepoNotAccessible is returned when a repository cannot be accessed
type ErrRepoNotAccessible struct {
	Repo   string
	Status int
}

func (e *ErrRepoNotAccessible) Error() string {
	return fmt.Sprintf("repository %s not accessible (status %d)", e.Repo, e.Status)
}

func FindProjectRoot(startDir string) (string, error) {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for currentDir != "/" {
		if _, err := os.Stat(filepath.Join(currentDir, ".git")); err == nil {
			return currentDir, nil
		}
		if _, err := os.Stat(filepath.Join(currentDir, ".github")); err == nil {
			return currentDir, nil
		}
		currentDir = filepath.Dir(currentDir)
	}

	return "", fmt.Errorf("could not find project root")
}

func FindActionReferences(startDir string) ([]ActionReference, error) {
	projectRoot, err := FindProjectRoot(startDir)
	if err != nil {
		return nil, err
	}

	workflowDir := filepath.Join(projectRoot, ".github", "workflows")
	actionRefs := []ActionReference{}
	seen := make(map[string]bool)

	err = filepath.Walk(workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !(strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var workflow map[string]interface{}
		if err := yaml.Unmarshal(content, &workflow); err != nil {
			return err
		}

		// Get relative path from project root
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		refs := extractActionUses(workflow)
		for _, ref := range refs {
			key := ref.Name + "@" + ref.Version + "@" + relPath
			if !seen[key] {
				seen[key] = true
				actionRefs = append(actionRefs, ActionReference{
					Name:    ref.Name,
					Version: ref.Version,
					File:    relPath,
				})
			}
		}

		return nil
	})

	return actionRefs, err
}

// extractActionUses recursively searches for "uses" fields in the workflow
func extractActionUses(obj interface{}) []ActionReference {
	refs := []ActionReference{}

	switch v := obj.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if key == "uses" {
				if uses, ok := val.(string); ok && strings.Contains(uses, "@") {
					// Skip local actions (./path/to/action)
					if strings.HasPrefix(uses, "./") {
						continue
					}
					parts := strings.SplitN(uses, "@", 2)
					if len(parts) == 2 {
						refs = append(refs, ActionReference{
							Name:    parts[0],
							Version: parts[1],
						})
					}
				}
			} else {
				refs = append(refs, extractActionUses(val)...)
			}
		}
	case []interface{}:
		for _, item := range v {
			refs = append(refs, extractActionUses(item)...)
		}
	}

	return refs
}

// CheckResult contains the results of checking action versions
type CheckResult struct {
	Outdated []OutdatedAction
	Warnings []string
}

func CheckActionVersions(actions []ActionReference) (bool, CheckResult, error) {
	result := CheckResult{}

	// Cache latest versions to avoid duplicate API calls (keyed by repo)
	latestVersionCache := make(map[string]string)
	skippedRepos := make(map[string]bool)

	for _, action := range actions {
		repo := repoFromAction(action.Name)

		// Skip if we already know this repo is inaccessible
		if skippedRepos[repo] {
			continue
		}

		latestVersion, ok := latestVersionCache[repo]
		if !ok {
			var err error
			latestVersion, err = fetchLatestMajorVersion(action)
			if err != nil {
				var notAccessible *ErrRepoNotAccessible
				if errors.As(err, &notAccessible) {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("skipping %s: repository not accessible", action.Name))
					skippedRepos[repo] = true
					continue
				}
				return false, result, fmt.Errorf("failed to check %s: %w", action.Name, err)
			}
			latestVersionCache[repo] = latestVersion
		}

		if !isUpToDate(action.Version, latestVersion) {
			result.Outdated = append(result.Outdated, OutdatedAction{
				Name:           action.Name,
				CurrentVersion: action.Version,
				LatestVersion:  latestVersion,
				File:           action.File,
			})
		}
	}

	return len(result.Outdated) == 0, result, nil
}

// repoFromAction extracts the owner/repo from an action name
// e.g., "actions/cache/restore" -> "actions/cache"
func repoFromAction(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return name
}

// fetchLatestMajorVersion fetches all tags from GitHub and returns the highest major version tag
func fetchLatestMajorVersion(action ActionReference) (string, error) {
	repo := repoFromAction(action.Name)

	// GitHub API URL for tags
	url := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=100", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Use GITHUB_TOKEN if available for higher rate limits
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return "", &ErrRepoNotAccessible{Repo: repo, Status: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var tags []GitHubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}

	// Find the highest major version tag (e.g., v6, v5, etc.)
	latestMajor := findLatestMajorVersionTag(tags)
	if latestMajor == "" {
		return action.Version, nil // No version tags found, assume current is fine
	}

	return latestMajor, nil
}

// versionTagRegex matches tags like v1, v2, v10, etc. (major version only)
var versionTagRegex = regexp.MustCompile(`^v(\d+)$`)

// findLatestMajorVersionTag finds the highest major version tag (v1, v2, etc.)
func findLatestMajorVersionTag(tags []GitHubTag) string {
	var majorVersions []int

	for _, tag := range tags {
		matches := versionTagRegex.FindStringSubmatch(tag.Name)
		if matches != nil {
			major, err := strconv.Atoi(matches[1])
			if err == nil {
				majorVersions = append(majorVersions, major)
			}
		}
	}

	if len(majorVersions) == 0 {
		return ""
	}

	sort.Sort(sort.Reverse(sort.IntSlice(majorVersions)))
	return fmt.Sprintf("v%d", majorVersions[0])
}

// isUpToDate checks if the current version is up to date with the latest
// For major versions (v1, v2), compares major numbers
// For SHA refs or other formats, returns true (can't compare)
func isUpToDate(current, latest string) bool {
	currentMajor := extractMajorVersion(current)
	latestMajor := extractMajorVersion(latest)

	// If we can't extract major versions, assume up to date
	if currentMajor == -1 || latestMajor == -1 {
		return true
	}

	return currentMajor >= latestMajor
}

// extractMajorVersion extracts the major version number from a version string
// Returns -1 if not a valid major version format
func extractMajorVersion(version string) int {
	// Match v1, v2, v10, v1.2.3, etc.
	re := regexp.MustCompile(`^v(\d+)`)
	matches := re.FindStringSubmatch(version)
	if matches == nil {
		return -1
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1
	}

	return major
}