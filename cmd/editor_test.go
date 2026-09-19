package cmd

import (
	"os"
	"reflect"
	"testing"
)

func TestGetEditorCommand(t *testing.T) {
	origVisual := os.Getenv("VISUAL")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("VISUAL", origVisual)
		os.Setenv("EDITOR", origEditor)
	}()

	t.Run("VISUAL takes precedence over EDITOR", func(t *testing.T) {
		os.Setenv("VISUAL", "code --wait")
		os.Setenv("EDITOR", "nano")

		bin, args := getEditorCommand()
		if bin != "code" {
			t.Errorf("bin = %q, expected 'code'", bin)
		}
		expectedArgs := []string{"--wait"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Errorf("args = %v, expected %v", args, expectedArgs)
		}
	})

	t.Run("EDITOR with multiple flags", func(t *testing.T) {
		os.Setenv("VISUAL", "")
		os.Setenv("EDITOR", "subl -n -w")

		bin, args := getEditorCommand()
		if bin != "subl" {
			t.Errorf("bin = %q, expected 'subl'", bin)
		}
		expectedArgs := []string{"-n", "-w"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Errorf("args = %v, expected %v", args, expectedArgs)
		}
	})

	t.Run("Fallback when neither is set", func(t *testing.T) {
		os.Setenv("VISUAL", "")
		os.Setenv("EDITOR", "")

		bin, args := getEditorCommand()
		if bin == "" {
			t.Errorf("expected default fallback editor, got empty")
		}
		if len(args) != 0 {
			t.Errorf("expected no extra args for fallback, got %v", args)
		}
	})
}
