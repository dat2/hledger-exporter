package cmd

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/plaid/plaid-go/plaid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// config
type then struct {
	Account2 string
}

type rule struct {
	If   string
	Then then
}

type config struct {
	Accounts         map[string]string
	Rules            []rule
	PlaidClientID    string `mapstructure:"PLAID_CLIENT_ID" toml:"-"`
	PlaidSecret      string `mapstructure:"PLAID_SECRET" toml:"-"`
	PlaidPublicKey   string `mapstructure:"PLAID_PUBLIC_KEY" toml:"-"`
	PlaidAccessToken string `mapstructure:"PLAID_ACCESS_TOKEN" toml:"-"`
}

var c config

// transaction
type hledgerAccount struct {
	Name   string
	Amount float64
}

type hledgerTransaction struct {
	Date        time.Time
	Description string
	Accounts    []hledgerAccount
}

func (t hledgerTransaction) String() string {
	// we need to find the padding to put between <account name> and amount
	longestAccountNameLength := 0
	for _, account := range t.Accounts {
		if len(account.Name) > longestAccountNameLength {
			longestAccountNameLength = len(account.Name)
		}
	}
	lines := make([]string, len(t.Accounts))
	for i, account := range t.Accounts {
		lines[i] = fmt.Sprintf("    %-*s    %.2f", longestAccountNameLength, account.Name, account.Amount)
	}
	return fmt.Sprintf("%s %s\n%s", t.Date.Format("2006-01-02"), t.Description, strings.Join(lines, "\n"))
}

var rootCmd = &cobra.Command{
	Use:   "hledger-exporter",
	Short: "Hledger Exporter will export your transactions in hledger format.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a date argument")
		}
		_, err := dateparse.ParseStrict(args[0])
		if err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// we've already validated startDate above.
		startDate, _ := dateparse.ParseStrict(args[0])
		endDate := time.Now()

		// compile regexs
		regexps := make([]*regexp.Regexp, len(c.Rules))
		for i, rule := range c.Rules {
			regexps[i] = regexp.MustCompile("(?i)" + rule.If)
		}

		// create a plaid client
		clientOptions := plaid.ClientOptions{
			c.PlaidClientID,
			c.PlaidSecret,
			c.PlaidPublicKey,
			plaid.Development,
			&http.Client{},
		}
		client, err := plaid.NewClient(clientOptions)
		if err != nil {
			panic(fmt.Errorf("Failed to initialize plaid client: %w", err))
		}

		// loop through all transactions in the page
		transactions := make([]plaid.Transaction, 0)
		totalTransactions := 0
		fetchedTransactions := 0
		for {
			page, err := client.GetTransactions(c.PlaidAccessToken, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
			if err != nil {
				panic(fmt.Errorf("Failed to load transactions: %w", err))
			}
			if totalTransactions == 0 {
				totalTransactions = page.TotalTransactions
			}
			fetchedTransactions += len(page.Transactions)
			transactions = append(transactions, page.Transactions...)
			if fetchedTransactions >= totalTransactions {
				break
			}
		}

		// convert to hledger transactions
		// even though we know the length of transactions, we don't know how many pending there are in advance.
		hledgerTransactions := make([]hledgerTransaction, 0)
		pendingTransactions := 0
		for _, transaction := range transactions {
			if transaction.Pending {
				pendingTransactions++
				continue
			}

			// viper converts the account keys in the config.toml file to lower case
			hledgerAccountName, ok := c.Accounts[strings.ToLower(transaction.AccountID)]
			if !ok {
				// warn on stderr
				continue
			}

			date, err := dateparse.ParseStrict(transaction.Date)
			if err != nil {
				panic(fmt.Errorf("Plaid returned an invalid date string: %w", err))
			}

			// match the rules
			account2Name := "expenses:unknown"
			for i, rule := range c.Rules {
				if regexps[i].MatchString(transaction.Name) {
					account2Name = rule.Then.Account2
				}
			}

			hledgerTransaction := hledgerTransaction{
				Date:        date,
				Description: transaction.Name,
				Accounts: []hledgerAccount{
					hledgerAccount{Name: hledgerAccountName, Amount: -transaction.Amount},
					hledgerAccount{Name: account2Name, Amount: transaction.Amount},
				},
			}
			hledgerTransactions = append(hledgerTransactions, hledgerTransaction)
		}

		for _, transaction := range hledgerTransactions {
			fmt.Printf("%s\n\n", transaction)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
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
	err = viper.Unmarshal(&c)
	if err != nil {
		panic(fmt.Errorf("Fatal error from config file: %w", err))
	}
}
