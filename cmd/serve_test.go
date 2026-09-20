package cmd

import (
	"testing"
)

func TestServeSubcommands(t *testing.T) {
	// Verify serveCmd exists and has subcommands
	if serveCmd == nil {
		t.Fatal("expected serveCmd to be defined")
	}

	foundHTTP := false
	foundMCP := false

	for _, c := range serveCmd.Commands() {
		if c.Name() == "http" {
			foundHTTP = true
			portFlag := c.Flags().Lookup("port")
			if portFlag == nil || portFlag.DefValue != "8080" {
				t.Errorf("expected default port 8080, got %v", portFlag)
			}
			hostFlag := c.Flags().Lookup("host")
			if hostFlag == nil || hostFlag.DefValue != "127.0.0.1" {
				t.Errorf("expected default host 127.0.0.1, got %v", hostFlag)
			}
			openFlag := c.Flags().Lookup("open")
			if openFlag == nil || openFlag.DefValue != "false" {
				t.Errorf("expected default open false, got %v", openFlag)
			}
		}
		if c.Name() == "mcp" {
			foundMCP = true
		}
	}

	if !foundHTTP {
		t.Error("expected 'http' subcommand under 'serve'")
	}
	if !foundMCP {
		t.Error("expected 'mcp' subcommand under 'serve'")
	}
}
