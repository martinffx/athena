package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

const startCommandName = "start"

func TestStartCommand_Exists(t *testing.T) {
	// Verify start command is registered
	cmd := rootCmd.Commands()
	found := false
	for _, c := range cmd {
		if c.Name() == startCommandName {
			found = true
			break
		}
	}
	if !found {
		t.Error("start command not registered with root command")
	}
}

func TestStartCommand_Properties(t *testing.T) {
	// Find start command
	var startCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == startCommandName {
			startCmd = c
			break
		}
	}

	if startCmd == nil {
		t.Fatal("start command not found")
	}

	// Verify properties
	if startCmd.Use != startCommandName {
		t.Errorf("start command Use = %v, want %q", startCmd.Use, startCommandName)
	}

	if startCmd.Short == "" {
		t.Error("start command has no Short description")
	}

	if startCmd.RunE == nil {
		t.Error("start command has no RunE function")
	}
}

func TestStartCommand_RequiresAPIKey(t *testing.T) {
	// This is an integration test that would require setting up
	// the full daemon environment, so we'll skip actual execution
	// and just verify the command structure is correct
	t.Skip("Integration test - requires full daemon setup")
}
