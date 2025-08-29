package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	scriptDir := filepath.Dir(execPath)

	projectDir := filepath.Join(scriptDir, "..", "..", "..", "..")
	devenvFile := filepath.Join(projectDir, "devenv.nix")
	if _, err := os.Stat(devenvFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: devenv.nix file not found in project directory\n")
		os.Exit(1)
	}

	goPath := filepath.Join(scriptDir, "go.orig")
	if _, err := os.Stat(goPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: go.orig executable not found in %s\n", scriptDir)
		os.Exit(1)
	}

	tmpFile, err := os.CreateTemp("", "go_wrapper_*.sh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
		os.Exit(1)
	}
	scriptPath := tmpFile.Name()
	tmpFile.Close()

	//fmt.Printf("Creating wrapper script at %s\n", scriptPath)
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current working directory: %v\n", err)
		os.Exit(1)
	}
	if err := createWrapperScript(scriptPath, goPath, os.Args[1:], pwd); err != nil {
		os.Remove(scriptPath)
		fmt.Fprintf(os.Stderr, "Error creating wrapper script: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("sh", scriptPath)
	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error executing wrapper script: %v\n", err)
		os.Exit(1)
	}

	os.Remove(scriptPath)
}

// quoteArgs properly quotes command-line arguments to handle spaces and special characters
func createWrapperScript(scriptPath, goPath string, args []string, pwd string) error {
	content := fmt.Sprintf(`#!/bin/sh
if [ ! -z "$DEVENV_ROOT" ]; then
    cd %s
    exec %s %s
fi
eval "$(direnv export bash 2>/dev/null)" >/dev/null 2>&1
cd %s
exec %s %s
`, pwd, goPath, strings.Join(quoteArgs(args), " "), pwd, goPath, strings.Join(quoteArgs(args), " "))
	return os.WriteFile(scriptPath, []byte(content), 0755)
}

func quoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		// If the argument contains spaces or special characters, quote it
		if strings.ContainsAny(arg, " \t\n\r\"'`$&*()[]{}|;<>?!") {
			quoted[i] = fmt.Sprintf("\"%s\"", strings.Replace(arg, "\"", "\\\"", -1))
		} else {
			quoted[i] = arg
		}
	}
	return quoted
}
