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
  help     Print this help message
  version  Print the version of aver
  --json   Output results as JSON

Check GitHub Actions versions in the current project.
Exits with status 0 if all actions are up to date,
or status 1 with a list of outdated actions.

Examples:
  aver          Check actions in current project
  aver --json   Output as JSON
  aver help     Show this help message`

func printHelp() {
	fmt.Println(usageText)
}

func printVersion() {
	fmt.Println("aver version 0.1.0")
}

func printTable(outdated []actions.OutdatedAction) {
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

func printJSON(outdated []actions.OutdatedAction) error {
	output, err := json.MarshalIndent(outdated, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
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

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	actionRefs, err := actions.FindActionReferences(dir)
	if err != nil {
		log.Fatal(err)
	}

	upToDate, result, err := actions.CheckActionVersions(actionRefs)
	if err != nil {
		log.Fatal(err)
	}

	// Print warnings to stderr
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}

	if upToDate {
		if jsonOutput {
			fmt.Println("[]")
		}
		os.Exit(0)
	}

	if jsonOutput {
		if err := printJSON(result.Outdated); err != nil {
			log.Fatal(err)
		}
	} else {
		printTable(result.Outdated)
	}
	os.Exit(1)
}