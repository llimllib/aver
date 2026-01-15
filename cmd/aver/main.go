package main

import (
	"fmt"
	"log"
	"os"

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

	// Print outdated actions table
	fmt.Println("Action | Current Version | Latest Version")
	fmt.Println("-------|-----------------|----------------")
	for _, action := range outdatedActions {
		fmt.Printf("%s | %s | %s\n", action.Name, action.CurrentVersion, action.LatestVersion)
	}

	os.Exit(1)
}