package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"aver/pkg/actions"
)

// Exit codes
const (
	exitOK       = 0
	exitOutdated = 1
	exitError    = 2
)

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(exitError)
}

const usageText = `aver: GitHub Actions version checker

Usage:
  aver [options]

Options:
  help           Print this help message
  version        Print the version of aver
  --json         Output results as JSON
  --ignore-sha   Ignore SHA-pinned actions
  --ignore-minor Only check major version differences
  --quiet        Suppress progress indicator

Check GitHub Actions versions in the current project.

Exit codes:
  0  All actions are up to date
  1  Outdated actions found
  2  Error occurred (e.g., network failure, invalid workflow)

Examples:
  aver                Check actions in current project
  aver --json         Output as JSON
  aver --ignore-sha   Ignore SHA-pinned actions
  aver --ignore-minor Only report major version updates
  aver --quiet        Run without progress indicator
  aver help           Show this help message`

func printHelp() {
	fmt.Println(usageText)
}

func printVersion() {
	fmt.Println("aver version 0.1.0")
}

func printOutdatedTable(outdated []actions.OutdatedAction) {
	if len(outdated) == 0 {
		return
	}

	// Column headers
	headers := []string{"File", "Action", "Current", "Latest"}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, a := range outdated {
		if len(a.File) > widths[0] {
			widths[0] = len(a.File)
		}
		if len(a.Name) > widths[1] {
			widths[1] = len(a.Name)
		}
		if len(a.CurrentVersion) > widths[2] {
			widths[2] = len(a.CurrentVersion)
		}
		if len(a.LatestVersion) > widths[3] {
			widths[3] = len(a.LatestVersion)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s  %-*s\n",
		widths[0], headers[0],
		widths[1], headers[1],
		widths[2], headers[2],
		widths[3], headers[3])

	// Print separator
	fmt.Printf("%s  %s  %s  %s\n",
		strings.Repeat("-", widths[0]),
		strings.Repeat("-", widths[1]),
		strings.Repeat("-", widths[2]),
		strings.Repeat("-", widths[3]))

	// Print rows
	for _, a := range outdated {
		fmt.Printf("%-*s  %-*s  %-*s  %-*s\n",
			widths[0], a.File,
			widths[1], a.Name,
			widths[2], a.CurrentVersion,
			widths[3], a.LatestVersion)
	}
}

func printSHATable(shaPinned []actions.SHAPinnedAction) {
	if len(shaPinned) == 0 {
		return
	}

	// Column headers
	headers := []string{"File", "Action", "Current SHA", "Latest SHA", "Behind"}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, a := range shaPinned {
		if len(a.File) > widths[0] {
			widths[0] = len(a.File)
		}
		if len(a.Name) > widths[1] {
			widths[1] = len(a.Name)
		}
		// Show short SHA (first 7 chars)
		currentShort := shortSHA(a.CurrentSHA)
		latestShort := shortSHA(a.LatestSHA)
		if len(currentShort) > widths[2] {
			widths[2] = len(currentShort)
		}
		if len(latestShort) > widths[3] {
			widths[3] = len(latestShort)
		}
		behindStr := fmt.Sprintf("%d", a.CommitsBehind)
		if len(behindStr) > widths[4] {
			widths[4] = len(behindStr)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*s\n",
		widths[0], headers[0],
		widths[1], headers[1],
		widths[2], headers[2],
		widths[3], headers[3],
		widths[4], headers[4])

	// Print separator
	fmt.Printf("%s  %s  %s  %s  %s\n",
		strings.Repeat("-", widths[0]),
		strings.Repeat("-", widths[1]),
		strings.Repeat("-", widths[2]),
		strings.Repeat("-", widths[3]),
		strings.Repeat("-", widths[4]))

	// Print rows
	for _, a := range shaPinned {
		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*d\n",
			widths[0], a.File,
			widths[1], a.Name,
			widths[2], shortSHA(a.CurrentSHA),
			widths[3], shortSHA(a.LatestSHA),
			widths[4], a.CommitsBehind)
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

type jsonOutput struct {
	Outdated  []actions.OutdatedAction  `json:"outdated"`
	SHAPinned []actions.SHAPinnedAction `json:"sha_pinned"`
}

func printJSON(result actions.CheckResult) error {
	output := jsonOutput{
		Outdated:  result.Outdated,
		SHAPinned: result.SHAPinned,
	}
	if output.Outdated == nil {
		output.Outdated = []actions.OutdatedAction{}
	}
	if output.SHAPinned == nil {
		output.SHAPinned = []actions.SHAPinnedAction{}
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func hasFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

// spinner displays a spinning progress indicator
type spinner struct {
	frames  []string
	stop    chan struct{}
	stopped chan struct{}
	action  chan string
	current string
}

func newSpinner() *spinner {
	return &spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
		action:  make(chan string),
	}
}

func (s *spinner) start() {
	go func() {
		defer close(s.stopped)
		i := 0
		for {
			select {
			case <-s.stop:
				// Clear the spinner
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			case name := <-s.action:
				s.current = name
			default:
				msg := "Checking actions..."
				if s.current != "" {
					msg = fmt.Sprintf("Checking %s...", s.current)
				}
				fmt.Fprintf(os.Stderr, "\r\033[K%s %s", s.frames[i%len(s.frames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *spinner) update(action string) {
	select {
	case s.action <- action:
	default:
		// Don't block if channel is full
	}
}

func (s *spinner) finish() {
	close(s.stop)
	<-s.stopped
}

// isTerminal returns true if the file is a terminal
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func main() {
	args := os.Args[1:]

	// Handle help and version flags
	if hasFlag(args, "help", "--help", "-h") {
		printHelp()
		os.Exit(0)
	}
	if hasFlag(args, "version", "--version", "-v") {
		printVersion()
		os.Exit(0)
	}

	jsonOutput := hasFlag(args, "--json", "-json", "json")
	ignoreSHA := hasFlag(args, "--ignore-sha", "-ignore-sha", "ignore-sha")
	ignoreMinor := hasFlag(args, "--ignore-minor", "-ignore-minor", "ignore-minor")
	quiet := hasFlag(args, "--quiet", "-quiet", "quiet", "-q")

	dir, err := os.Getwd()
	if err != nil {
		fatal(err.Error())
	}

	actionRefs, err := actions.FindActionReferences(dir)
	if err != nil {
		fatal(err.Error())
	}

	opts := actions.CheckOptions{
		IgnoreSHA:   ignoreSHA,
		IgnoreMinor: ignoreMinor,
	}

	// Start spinner unless quiet mode, JSON output, or non-TTY stderr
	var spin *spinner
	if !quiet && !jsonOutput && isTerminal(os.Stderr) {
		spin = newSpinner()
		opts.OnProgress = spin.update
		spin.start()
	}

	upToDate, result, err := actions.CheckActionVersions(actionRefs, opts)

	// Stop spinner before any output
	if spin != nil {
		spin.finish()
	}
	if err != nil {
		fatal(err.Error())
	}

	// Print warnings to stderr
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}

	if upToDate {
		if jsonOutput {
			if err := printJSON(result); err != nil {
				fatal(err.Error())
			}
		}
		os.Exit(exitOK)
	}

	if jsonOutput {
		if err := printJSON(result); err != nil {
			fatal(err.Error())
		}
	} else {
		if len(result.Outdated) > 0 {
			fmt.Println("Outdated actions:")
			printOutdatedTable(result.Outdated)
		}
		if len(result.SHAPinned) > 0 {
			if len(result.Outdated) > 0 {
				fmt.Println()
			}
			fmt.Println("SHA-pinned actions behind default branch:")
			printSHATable(result.SHAPinned)
		}
	}
	os.Exit(exitOutdated)
}