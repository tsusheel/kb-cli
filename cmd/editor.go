package cmd

import (
	"os"
	"os/exec"
)

func captureEditorContent(initialContent string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Fallback to vi for linux 
	}

	f, err := os.CreateTemp("", "kb-note-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	
	if initialContent != "" {
		if _, err := f.WriteString(initialContent); err != nil {
			f.Close()
			return "", err
		}
	}
	f.Close()

	cmd := exec.Command(editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}

	return string(content), nil
}
