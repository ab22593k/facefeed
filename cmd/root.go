package cmd

import (
	theme "facefeed/internal"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// client is the shared HTTP client used across commands.
var client *http.Client

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "facefeed",
	Short: "Robust CLI Facebook publishing tool",
	Long: `facefeed is a CLI tool for publishing content to Facebook Pages.
It supports text posts, links, and image uploads (including automatic SVG-to-PNG conversion).
Content can be published to multiple targets (pages and groups) in a single run.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		client = &http.Client{Timeout: timeout}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
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
