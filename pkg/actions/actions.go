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

type SHAPinnedAction struct {
	File          string `json:"file"`
	Name          string `json:"action"`
	CurrentSHA    string `json:"current_sha"`
	LatestSHA     string `json:"latest_sha"`
	CommitsBehind int    `json:"commits_behind"`
}

// GitHubTag represents a tag from the GitHub API
type GitHubTag struct {
	Name string `json:"name"`
}

// GitHubCompare represents the compare API response
type GitHubCompare struct {
	AheadBy  int    `json:"ahead_by"`
	BehindBy int    `json:"behind_by"`
	Status   string `json:"status"`
}

// GitHubRepo represents repository info from the API
type GitHubRepo struct {
	DefaultBranch string `json:"default_branch"`
}

// GitHubRef represents a git reference from the API
type GitHubRef struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
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

// CheckOptions configures the behavior of CheckActionVersions
type CheckOptions struct {
	IgnoreSHA   bool
	IgnoreMinor bool
}

// CheckResult contains the results of checking action versions
type CheckResult struct {
	Outdated  []OutdatedAction
	SHAPinned []SHAPinnedAction
	Warnings  []string
}

// tagCache stores fetched tags per repo
type tagCache struct {
	tags map[string][]GitHubTag
}

func newTagCache() *tagCache {
	return &tagCache{tags: make(map[string][]GitHubTag)}
}

func (tc *tagCache) getTags(repo string) ([]GitHubTag, error) {
	if tags, ok := tc.tags[repo]; ok {
		return tags, nil
	}

	tags, err := fetchTags(repo)
	if err != nil {
		return nil, err
	}

	tc.tags[repo] = tags
	return tags, nil
}

func CheckActionVersions(actions []ActionReference, opts CheckOptions) (bool, CheckResult, error) {
	result := CheckResult{}
	cache := newTagCache()
	skippedRepos := make(map[string]bool)

	for _, action := range actions {
		repo := repoFromAction(action.Name)

		// Skip if we already know this repo is inaccessible
		if skippedRepos[repo] {
			continue
		}

		// Check if this is a SHA-pinned action
		if isSHA(action.Version) {
			if opts.IgnoreSHA {
				continue
			}

			// Check how far behind the SHA is
			shaInfo, err := checkSHAStatus(repo, action.Version)
			if err != nil {
				var notAccessible *ErrRepoNotAccessible
				if errors.As(err, &notAccessible) {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("skipping %s: repository not accessible", action.Name))
					skippedRepos[repo] = true
					continue
				}
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("skipping %s: %v", action.Name, err))
				continue
			}

			if shaInfo.CommitsBehind > 0 {
				result.SHAPinned = append(result.SHAPinned, SHAPinnedAction{
					File:          action.File,
					Name:          action.Name,
					CurrentSHA:    action.Version,
					LatestSHA:     shaInfo.LatestSHA,
					CommitsBehind: shaInfo.CommitsBehind,
				})
			}
			continue
		}

		tags, err := cache.getTags(repo)
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

		latestVersion := findLatestVersion(tags, action.Version, opts.IgnoreMinor)
		if latestVersion == "" {
			continue // No comparable version found
		}

		if !versionsEqual(action.Version, latestVersion) {
			result.Outdated = append(result.Outdated, OutdatedAction{
				Name:           action.Name,
				CurrentVersion: action.Version,
				LatestVersion:  latestVersion,
				File:           action.File,
			})
		}
	}

	allUpToDate := len(result.Outdated) == 0 && len(result.SHAPinned) == 0
	return allUpToDate, result, nil
}

// semver represents a parsed semantic version
type semver struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// parseSemver parses a version string into a semver struct
// Supports: v1, v1.2, v1.2.3
func parseSemver(version string) *semver {
	re := regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?$`)
	matches := re.FindStringSubmatch(version)
	if matches == nil {
		return nil
	}

	sv := &semver{Raw: version}

	sv.Major, _ = strconv.Atoi(matches[1])
	if matches[2] != "" {
		sv.Minor, _ = strconv.Atoi(matches[2])
	}
	if matches[3] != "" {
		sv.Patch, _ = strconv.Atoi(matches[3])
	}

	return sv
}

// compare returns -1 if s < other, 0 if equal, 1 if s > other
func (s *semver) compare(other *semver) int {
	if s.Major != other.Major {
		if s.Major < other.Major {
			return -1
		}
		return 1
	}
	if s.Minor != other.Minor {
		if s.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if s.Patch != other.Patch {
		if s.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// findLatestVersion finds the latest version tag
// If ignoreMinor is true, only compares major versions
// Otherwise, finds the latest version overall
func findLatestVersion(tags []GitHubTag, currentVersion string, ignoreMinor bool) string {
	currentSV := parseSemver(currentVersion)
	if currentSV == nil {
		return "" // Can't parse current version
	}

	var candidates []*semver
	for _, tag := range tags {
		sv := parseSemver(tag.Name)
		if sv == nil {
			continue
		}
		candidates = append(candidates, sv)
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates by version descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].compare(candidates[j]) > 0
	})

	if ignoreMinor {
		// Find the latest major version tag (just vN format)
		var latestMajor *semver
		for _, sv := range candidates {
			// Only consider pure major version tags (v1, v2, etc.)
			if sv.Minor == 0 && sv.Patch == 0 && strings.HasPrefix(sv.Raw, "v") && !strings.Contains(sv.Raw, ".") {
				if latestMajor == nil || sv.Major > latestMajor.Major {
					latestMajor = sv
				}
			}
		}
		if latestMajor != nil && latestMajor.Major > currentSV.Major {
			return latestMajor.Raw
		}
		return ""
	}

	// Find the latest version overall
	latest := candidates[0]
	if latest.compare(currentSV) > 0 {
		return latest.Raw
	}

	return ""
}

// versionsEqual checks if two version strings represent the same version
func versionsEqual(v1, v2 string) bool {
	sv1 := parseSemver(v1)
	sv2 := parseSemver(v2)
	if sv1 == nil || sv2 == nil {
		return v1 == v2
	}
	return sv1.compare(sv2) == 0
}

// isSHA returns true if the version string looks like a git SHA
func isSHA(version string) bool {
	// SHA commits are 40 hex characters (full) or 7+ hex characters (short)
	if len(version) < 7 {
		return false
	}
	for _, c := range version {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

type shaStatus struct {
	LatestSHA     string
	CommitsBehind int
}

// checkSHAStatus checks how far behind a SHA-pinned action is from the default branch
func checkSHAStatus(repo, sha string) (*shaStatus, error) {
	// First, get the default branch
	defaultBranch, err := getDefaultBranch(repo)
	if err != nil {
		return nil, err
	}

	// Get the latest SHA on the default branch
	latestSHA, err := getBranchHead(repo, defaultBranch)
	if err != nil {
		return nil, err
	}

	// If already at latest, no need to compare
	if strings.HasPrefix(latestSHA, sha) || strings.HasPrefix(sha, latestSHA) {
		return &shaStatus{LatestSHA: latestSHA, CommitsBehind: 0}, nil
	}

	// Compare the commits
	behindBy, err := compareCommits(repo, sha, defaultBranch)
	if err != nil {
		return nil, err
	}

	return &shaStatus{LatestSHA: latestSHA, CommitsBehind: behindBy}, nil
}

func getDefaultBranch(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
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

	var repoInfo GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", err
	}

	return repoInfo.DefaultBranch, nil
}

func getBranchHead(repo, branch string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/ref/heads/%s", repo, branch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ref GitHubRef
	if err := json.NewDecoder(resp.Body).Decode(&ref); err != nil {
		return "", err
	}

	return ref.Object.SHA, nil
}

func compareCommits(repo, baseSHA, head string) (int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/compare/%s...%s", repo, baseSHA, head)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var compare GitHubCompare
	if err := json.NewDecoder(resp.Body).Decode(&compare); err != nil {
		return 0, err
	}

	return compare.AheadBy, nil
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

// fetchTags fetches all tags from GitHub for a repository
func fetchTags(repo string) ([]GitHubTag, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=100", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil, &ErrRepoNotAccessible{Repo: repo, Status: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var tags []GitHubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	return tags, nil
}