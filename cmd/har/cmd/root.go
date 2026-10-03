package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	// Build-time version information injected by GoReleaser through -ldflags.
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "har",
	Short: "HAR (HTTP Archive) analysis tool",
	Long: `HAR Skills CLI — AI Agent Skill for HAR File Analysis

Parse, filter, and analyze HAR files; audit security, score performance, redact data,
transform requests, compare captures, merge and split files, and export results.

Examples:
  har -f capture.har info              # Show a HAR file summary
  har -f capture.har find "api/users"  # Search requests
  har -f capture.har security          # Run a security audit
  har -f capture.har performance       # Score performance
  har -f capture.har export curl       # Export as curl commands`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Customize the version output format.
	rootCmd.SetVersionTemplate(`HAR Skills {{.Version}}
commit:  {{.Annotations.commit}}
date:    {{.Annotations.date}}
`)
	rootCmd.Annotations = map[string]string{
		"commit": commit,
		"date":   date,
	}

	rootCmd.PersistentFlags().StringP("file", "f", "", "Path to the HAR file (use - to read stdin)")
	rootCmd.PersistentFlags().String("format", "text", "Output format (text, json, csv, yaml)")
	rootCmd.PersistentFlags().StringP("output", "o", "", "Output file path")
	rootCmd.PersistentFlags().Bool("no-header", false, "Hide table headers in text/CSV output")

	_ = viper.BindPFlag("file", rootCmd.PersistentFlags().Lookup("file"))
	_ = viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("no-header", rootCmd.PersistentFlags().Lookup("no-header"))

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default: $HOME/.har.yaml)")
}

func initConfig() {
	viper.SetEnvPrefix("HAR")
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".har")
		viper.AddConfigPath("$HOME")
		viper.AddConfigPath(".")
	}

	_ = viper.ReadInConfig()
}
