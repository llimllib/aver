package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"aver/pkg/actions"
)

const usageText = `aver: GitHub Actions version checker

Usage:
  aver [options]

Options:
  help         Print this help message
  version      Print the version of aver
  --json       Output results as JSON
  --ignore-sha Ignore SHA-pinned actions

Check GitHub Actions versions in the current project.
Exits with status 0 if all actions are up to date,
or status 1 with a list of outdated actions.

Examples:
  aver              Check actions in current project
  aver --json       Output as JSON
  aver --ignore-sha Ignore SHA-pinned actions
  aver help         Show this help message`

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

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	actionRefs, err := actions.FindActionReferences(dir)
	if err != nil {
		log.Fatal(err)
	}

	opts := actions.CheckOptions{
		IgnoreSHA: ignoreSHA,
	}

	upToDate, result, err := actions.CheckActionVersions(actionRefs, opts)
	if err != nil {
		log.Fatal(err)
	}

	// Print warnings to stderr
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}

	if upToDate {
		if jsonOutput {
			if err := printJSON(result); err != nil {
				log.Fatal(err)
			}
		}
		os.Exit(0)
	}

	if jsonOutput {
		if err := printJSON(result); err != nil {
			log.Fatal(err)
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
	os.Exit(1)
}