package cli

import (
	"testing"
)

func TestNewCaptureCommand(t *testing.T) {
	cmd := NewCaptureCommand()

	if cmd.Use != "capture" {
		t.Errorf("Command.Use = %v, want %v", cmd.Use, "capture")
	}

	if cmd.Short == "" {
		t.Error("Command.Short should not be empty")
	}

	// Check that required flags are defined
	flags := []string{"tool", "project", "tags", "auto-detect"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Flag %q not defined", flag)
		}
	}
}