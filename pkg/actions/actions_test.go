package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input    string
		expected *semver
	}{
		{"v1", &semver{Major: 1, Minor: 0, Patch: 0, Raw: "v1", HasMinor: false, HasPatch: false}},
		{"v1.2", &semver{Major: 1, Minor: 2, Patch: 0, Raw: "v1.2", HasMinor: true, HasPatch: false}},
		{"v1.2.3", &semver{Major: 1, Minor: 2, Patch: 3, Raw: "v1.2.3", HasMinor: true, HasPatch: true}},
		{"1.2.3", &semver{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3", HasMinor: true, HasPatch: true}},
		{"v10.20.30", &semver{Major: 10, Minor: 20, Patch: 30, Raw: "v10.20.30", HasMinor: true, HasPatch: true}},
		{"invalid", nil},
		{"v1.2.3.4", nil},
		{"vABC", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSemver(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Errorf("expected %+v, got nil", tt.expected)
				return
			}
			if result.Major != tt.expected.Major ||
				result.Minor != tt.expected.Minor ||
				result.Patch != tt.expected.Patch ||
				result.Raw != tt.expected.Raw ||
				result.HasMinor != tt.expected.HasMinor ||
				result.HasPatch != tt.expected.HasPatch {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestSemverCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v2.0.0", -1},
		{"v2.0.0", "v1.0.0", 1},
		{"v1.1.0", "v1.2.0", -1},
		{"v1.2.0", "v1.1.0", 1},
		{"v1.0.1", "v1.0.2", -1},
		{"v1.0.2", "v1.0.1", 1},
		{"v1", "v1.0.0", 0},
		{"v1.2", "v1.2.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			sv1 := parseSemver(tt.v1)
			sv2 := parseSemver(tt.v2)
			result := sv1.compare(sv2)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestIsSHA(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc1234", true},
		{"abcdef1234567890abcdef1234567890abcdef12", true}, // 40 chars
		{"ABC1234", true},
		{"1234567", true},
		{"abc123", false},  // too short
		{"ghijkl", false},  // invalid hex
		{"v1.0.0", false},  // version string
		{"abc123!", false}, // invalid char
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isSHA(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRepoFromAction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"actions/checkout", "actions/checkout"},
		{"actions/cache/restore", "actions/cache"},
		{"owner/repo/sub/path", "owner/repo"},
		{"singleword", "singleword"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := repoFromAction(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestVersionsEqual(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"v1", "v1.0.0", true},
		{"v1.2", "v1.2.0", true},
		{"v1.2.3", "v1.2.3", true},
		{"v1", "v2", false},
		{"v1.0.0", "v1.0.1", false},
		{"invalid", "invalid", true},
		{"invalid", "v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.v1+" == "+tt.v2, func(t *testing.T) {
			result := versionsEqual(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFindLatestVersion(t *testing.T) {
	tags := []GitHubTag{
		{Name: "v1.0.0"},
		{Name: "v1.0.1"},
		{Name: "v1.1.0"},
		{Name: "v1.2.0"},
		{Name: "v2.0.0"},
		{Name: "v2.1.0"},
		{Name: "v2.1.1"},
		{Name: "v1"},
		{Name: "v2"},
	}

	tests := []struct {
		name        string
		current     string
		ignoreMinor bool
		expected    string
	}{
		// Major-only versions should only report newer majors
		{"major only v1 -> v2 available", "v1", false, "v2.1.1"},
		{"major only v2 -> nothing newer", "v2", false, ""},

		// Major.minor versions should report newer minors (same major) or newer majors
		{"v1.0 -> v1.2 available", "v1.0", false, "v2.1.1"},
		{"v1.1 -> v1.2 available", "v1.1", false, "v2.1.1"},
		{"v2.0 -> v2.1 available", "v2.0", false, "v2.1.1"},
		{"v2.1 -> nothing newer", "v2.1", false, ""},

		// Full versions should report any newer version
		{"v1.0.0 -> v2.1.1 available", "v1.0.0", false, "v2.1.1"},
		{"v2.1.0 -> v2.1.1 available", "v2.1.0", false, "v2.1.1"},
		{"v2.1.1 -> nothing newer", "v2.1.1", false, ""},

		// ignoreMinor flag tests
		{"ignore minor from v1", "v1", true, "v2"},
		{"ignore minor from v2", "v2", true, ""},
		{"ignore minor from v1.0.0", "v1.0.0", true, "v2"},
		{"ignore minor already latest", "v2.1.0", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLatestVersion(tags, tt.current, tt.ignoreMinor)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractActionUses(t *testing.T) {
	workflow := map[string]interface{}{
		"jobs": map[string]interface{}{
			"build": map[string]interface{}{
				"steps": []interface{}{
					map[string]interface{}{
						"uses": "actions/checkout@v4",
					},
					map[string]interface{}{
						"uses": "actions/setup-go@v5",
					},
					map[string]interface{}{
						"run": "echo hello",
					},
					map[string]interface{}{
						"uses": "./local/action",
					},
				},
			},
		},
	}

	refs := extractActionUses(workflow)

	if len(refs) != 2 {
		t.Errorf("expected 2 refs, got %d", len(refs))
	}

	expected := []ActionReference{
		{Name: "actions/checkout", Version: "v4"},
		{Name: "actions/setup-go", Version: "v5"},
	}

	for _, exp := range expected {
		found := false
		for _, ref := range refs {
			if ref.Name == exp.Name && ref.Version == exp.Version {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find %s@%s", exp.Name, exp.Version)
		}
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "aver-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create .github directory
	githubDir := filepath.Join(tmpDir, ".github")
	if err := os.Mkdir(githubDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create nested directory
	nestedDir := filepath.Join(tmpDir, "src", "pkg")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test finding root from nested dir
	root, err := FindProjectRoot(nestedDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected %s, got %s", tmpDir, root)
	}

	// Test from root itself
	root, err = FindProjectRoot(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected %s, got %s", tmpDir, root)
	}

	// Test with no project root
	emptyDir, err := os.MkdirTemp("", "aver-empty")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(emptyDir) }()

	_, err = FindProjectRoot(emptyDir)
	if err == nil {
		t.Error("expected error for directory without project root")
	}
}

func TestTagCache(t *testing.T) {
	cache := newTagCache()

	// Manually populate cache
	cache.tags["owner/repo"] = []GitHubTag{
		{Name: "v1.0.0"},
		{Name: "v2.0.0"},
	}

	// Should return cached value
	tags, err := cache.getTags("owner/repo")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestErrRepoNotAccessible(t *testing.T) {
	err := &ErrRepoNotAccessible{Repo: "owner/repo", Status: 404}
	expected := "repository owner/repo not accessible (status 404)"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
