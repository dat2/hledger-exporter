package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// config
type Then struct {
	Account2 string
}

type Rule struct {
	If   string
	Then Then
}

type Config struct {
	Accounts         map[string]string
	Rules            []Rule
	PlaidClientID    string `mapstructure:"PLAID_CLIENT_ID" toml:"-"`
	PlaidSecret      string `mapstructure:"PLAID_SECRET" toml:"-"`
	PlaidPublicKey   string `mapstructure:"PLAID_PUBLIC_KEY" toml:"-"`
	PlaidAccessToken string `mapstructure:"PLAID_ACCESS_TOKEN" toml:"-"`
}

var config Config

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hledger-exporter",
		Short: "Hledger Exporter will export your transactions in hledger format.",
	}
	cobra.OnInitialize(initViperConfig)
	cmd.AddCommand(NewExportCmd(&config))
	cmd.AddCommand(NewPlaidCmd(&config))
	return cmd
}

func initViperConfig() {
	// set the file name that we read from
	viper.SetConfigName("config")

	// read from $HOME, $CWD
	viper.AddConfigPath("$HOME/.hledger-exporter")
	viper.AddConfigPath(".")

	// let viper know that it has to look for these things in the environment
	// https://github.com/spf13/viper/issues/188
	viper.AutomaticEnv()
	viper.BindEnv("PLAID_CLIENT_ID")
	viper.BindEnv("PLAID_SECRET")
	viper.BindEnv("PLAID_PUBLIC_KEY")
	viper.BindEnv("PLAID_ACCESS_TOKEN")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error from config file: %w", err))
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("Fatal error from config file: %w", err))
	}
}
