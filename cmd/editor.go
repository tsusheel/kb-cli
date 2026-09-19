package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// getEditorCommand returns the editor executable and any initial flags configured in the environment.
func getEditorCommand() (string, []string) {
	editorEnv := os.Getenv("VISUAL")
	if editorEnv == "" {
		editorEnv = os.Getenv("EDITOR")
	}
	editorEnv = strings.TrimSpace(editorEnv)

	if editorEnv == "" {
		if runtime.GOOS == "windows" {
			return "notepad", nil
		}
		return "vi", nil
	}

	parts := strings.Fields(editorEnv)
	if len(parts) == 0 {
		if runtime.GOOS == "windows" {
			return "notepad", nil
		}
		return "vi", nil
	}

	return parts[0], parts[1:]
}

func captureEditorContent(initialContent string) (string, error) {
	bin, args := getEditorCommand()

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

	cmdArgs := append(args, f.Name())
	cmd := exec.Command(bin, cmdArgs...)
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
