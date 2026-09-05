package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/mcp"
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"mcp"},
	Short:   "Start MCP (Model Context Protocol) server over stdio for AI agent connectivity",
	Long: `Start the MCP server over standard input/output (stdio).

This allows AI assistants and agents (Claude Desktop, Antigravity, Cursor, Cline, etc.)
to list, filter, search, create, update, and manage your knowledge base notes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcp.StartServer()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
