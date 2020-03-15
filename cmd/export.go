package cmd

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/plaid/plaid-go/plaid"
	"github.com/spf13/cobra"
)

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

func NewExportCmd(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export transactions in hledger format.",
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// we've already validated startDate above.
			startDate, _ := dateparse.ParseStrict(args[0])
			endDate := time.Now()

			// compile regexs
			regexps := make([]*regexp.Regexp, len(config.Rules))
			for i, rule := range config.Rules {
				regexps[i] = regexp.MustCompile("(?i)" + rule.If)
			}

			// create a plaid client
			clientOptions := plaid.ClientOptions{
				config.PlaidClientID,
				config.PlaidSecret,
				config.PlaidPublicKey,
				plaid.Development,
				&http.Client{},
			}
			client, err := plaid.NewClient(clientOptions)
			if err != nil {
				return fmt.Errorf("Failed to initialize plaid client: %w", err)
			}

			// loop through all transactions in the page
			transactions := make([]plaid.Transaction, 0)
			totalTransactions := 0
			offset := 0
			for {
				page, err := client.GetTransactionsWithOptions(config.PlaidAccessToken,
					plaid.GetTransactionsOptions{
						StartDate:  startDate.Format("2006-01-02"),
						EndDate:    endDate.Format("2006-01-02"),
						AccountIDs: []string{},
						Count:      100,
						Offset:     offset,
					})
				if err != nil {
					return fmt.Errorf("Failed to load transactions: %w", err)
				}
				if totalTransactions == 0 {
					totalTransactions = page.TotalTransactions
				}
				offset += len(page.Transactions)
				transactions = append(transactions, page.Transactions...)
				if offset >= totalTransactions {
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
				hledgerAccountName, ok := config.Accounts[strings.ToLower(transaction.AccountID)]
				if !ok {
					// warn on stderr
					continue
				}

				date, err := dateparse.ParseStrict(transaction.Date)
				if err != nil {
					return fmt.Errorf("Plaid returned an invalid date string: %w", err)
				}

				// match the rules
				account2Name := "expenses:unknown"
				for i, rule := range config.Rules {
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

			sort.Slice(hledgerTransactions, func(i, j int) bool {
				return hledgerTransactions[i].Date.Before(hledgerTransactions[j].Date)
			})

			for _, transaction := range hledgerTransactions {
				fmt.Printf("%s\n\n", transaction)
			}

			return nil
		},
	}
}
