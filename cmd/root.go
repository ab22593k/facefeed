// Package cmd defines the CLI commands for publishing content to Facebook Pages.
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	theme "facefeed/internal"
	"facefeed/internal/facebook"

	"github.com/spf13/cobra"
)

// FBClient is the shared Facebook API client used across commands.
// It is initialized in PersistentPreRun and available to all subcommands.
var FBClient facebook.Client

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "facefeed",
	Short: "Robust CLI Facebook publishing tool",
	Long: `facefeed is a CLI tool for publishing content to Facebook Pages.
It supports text posts, links, and image uploads (including automatic SVG-to-PNG conversion).
Content can be published to multiple targets (pages and groups) in a single run.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		httpClient := &http.Client{Timeout: timeout}
		token := os.Getenv("FB_ACCESS_TOKEN")
		FBClient = facebook.New(httpClient, token)
	},
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	theme.PrintHeader()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Duration("timeout", 60*time.Second, "HTTP request timeout")
}
