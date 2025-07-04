package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	RootCmd.SetErr(&buf)
	RootCmd.SetArgs([]string{"--help"})

	err := RootCmd.Execute()
	if err != nil {
		t.Errorf("root command with --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "A simple CLI tool with version information") {
		t.Errorf("expected help output to contain description, got: %s", output)
	}
}

func TestRootCommandWithoutArgs(t *testing.T) {
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	RootCmd.SetErr(&buf)
	RootCmd.SetArgs([]string{})

	err := RootCmd.Execute()
	if err != nil {
		t.Errorf("root command without args failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "A simple CLI tool with version information") {
		t.Errorf("expected output to contain description, got: %s", output)
	}
}

func TestRootCommandHasSubcommands(t *testing.T) {
	commands := RootCmd.Commands()

	expectedCommands := []string{"version", "dump", "plan"}
	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Name()
	}

	for _, expected := range expectedCommands {
		found := false
		for _, actual := range commandNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %s not found in: %v", expected, commandNames)
		}
	}
}
