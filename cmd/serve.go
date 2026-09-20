package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/mcp"
	"github.com/tsusheel/kb-cli/server"
)

var (
	serveHTTPPort int
	serveHTTPHost string
	serveHTTPOpen bool
)

var serveCmd = &cobra.Command{
	Use:     "serve [subcommand]",
	Aliases: []string{"mcp"},
	Short:   "Start MCP server (stdio) or Web Fuzzy Finder HTTP server",
	Long: `Start server services for kb-cli.

Subcommands:
  kb serve http   Start the interactive web-based fuzzy finder with live preview pane
  kb serve mcp    Start the Model Context Protocol (MCP) server over stdio for AI agents

Running 'kb serve' without a subcommand defaults to starting the MCP stdio server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to MCP server for backward compatibility
		return mcp.StartServer()
	},
}

var serveMCPCmd = &cobra.Command{
	Use:     "mcp",
	Aliases: []string{"stdio"},
	Short:   "Start MCP (Model Context Protocol) server over stdio for AI agent connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcp.StartServer()
	},
}

var serveHTTPCmd = &cobra.Command{
	Use:     "http",
	Aliases: []string{"web", "ui"},
	Short:   "Start a modern web-based fuzzy finder with live preview pane",
	Long: `Start a lightweight embedded web server hosting an interactive fuzzy finder
with a dual-pane live markdown preview, real-time filtering, and keyboard navigation.

Examples:
  kb serve http
  kb serve http --port 3000 --open
  kb serve web -p 8080 -o`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := serveHTTPPort
		if !cmd.Flags().Changed("port") && viper.IsSet("server.port") {
			port = viper.GetInt("server.port")
		}
		host := serveHTTPHost
		if !cmd.Flags().Changed("host") && viper.IsSet("server.host") {
			host = viper.GetString("server.host")
		}
		cfg := server.Config{
			Host:        host,
			Port:        port,
			OpenBrowser: serveHTTPOpen,
		}
		return server.Start(cfg)
	},
}

func init() {
	serveHTTPCmd.Flags().IntVarP(&serveHTTPPort, "port", "p", 8080, "Port to listen on")
	serveHTTPCmd.Flags().StringVar(&serveHTTPHost, "host", "127.0.0.1", "Host address to bind to")
	serveHTTPCmd.Flags().BoolVarP(&serveHTTPOpen, "open", "o", false, "Automatically open web finder in default browser")

	serveCmd.AddCommand(serveMCPCmd)
	serveCmd.AddCommand(serveHTTPCmd)
	rootCmd.AddCommand(serveCmd)
}
