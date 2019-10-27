package cmd

import (
	"fmt"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "hledger-exporter",
	Short: "Hledger Exporter will export your transactions in hledger format.",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
	}

	// take config from ~/.hledger-export/config.toml
	viper.AddConfigPath(path.Join(home, ".hledger-export"))

	// support the CWD
	viper.AddConfigPath(".")

	// read config.toml from any directory
	viper.SetConfigName("config.toml")

	// take config from environment variables
	viper.SetEnvPrefix("hledger_export")
	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
