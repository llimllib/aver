package main

import (
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

Check GitHub Actions versions in the current project.
Exits with status 0 if all actions are up to date,
or status 1 with a list of outdated actions.

Examples:
  aver          Check actions in current project
  aver help     Show this help message`

func printHelp() {
	fmt.Println(usageText)
}

func printVersion() {
	fmt.Println("aver version 0.1.0")
}

func printTable(outdated []actions.OutdatedAction) {
	// Column headers
	headers := []string{"Action", "Current", "Latest"}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, a := range outdated {
		if len(a.Name) > widths[0] {
			widths[0] = len(a.Name)
		}
		if len(a.CurrentVersion) > widths[1] {
			widths[1] = len(a.CurrentVersion)
		}
		if len(a.LatestVersion) > widths[2] {
			widths[2] = len(a.LatestVersion)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s\n",
		widths[0], headers[0],
		widths[1], headers[1],
		widths[2], headers[2])

	// Print separator
	fmt.Printf("%s  %s  %s\n",
		strings.Repeat("-", widths[0]),
		strings.Repeat("-", widths[1]),
		strings.Repeat("-", widths[2]))

	// Print rows
	for _, a := range outdated {
		fmt.Printf("%-*s  %-*s  %-*s\n",
			widths[0], a.Name,
			widths[1], a.CurrentVersion,
			widths[2], a.LatestVersion)
	}
}

func main() {
	// Handle help and version flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "help", "--help", "-h":
			printHelp()
			os.Exit(0)
		case "version", "--version", "-v":
			printVersion()
			os.Exit(0)
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	actionRefs, err := actions.FindActionReferences(dir)
	if err != nil {
		log.Fatal(err)
	}

	upToDate, outdatedActions, err := actions.CheckActionVersions(actionRefs)
	if err != nil {
		log.Fatal(err)
	}

	if upToDate {
		os.Exit(0)
	}

	printTable(outdatedActions)
	os.Exit(1)
}