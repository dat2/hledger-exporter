package cmd

import (
	"fmt"
	"github.com/markbates/pkger"
	"github.com/plaid/plaid-go/plaid"
	"github.com/spf13/cobra"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

func NewPlaidCmd(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update plaid item when access token expires.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// create a plaid client
			clientOptions := plaid.ClientOptions{
				config.PlaidClientID,
				config.PlaidSecret,
				plaid.Development,
				&http.Client{},
			}

			client, err := plaid.NewClient(clientOptions)
			if err != nil {
				return fmt.Errorf("Failed to initialize plaid client: %w", err)
			}

			// create a public token
			resp, err := client.CreatePublicToken(config.PlaidAccessToken)
			if err != nil {
				return fmt.Errorf("Failed to create a public token: %w", err)
			}

			// load and parse template
			f, err := pkger.Open("/templates/update.html")
			if err != nil {
				return err
			}
			defer f.Close()

			contents, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}

			updateTemplate, err := template.New("update.html").Parse(string(contents))
			if err != nil {
				return err
			}

			// render the template
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				data := struct {
					PlaidClientID    string
					PlaidPublicToken string
				}{
					PlaidClientID:    config.PlaidClientID,
					PlaidPublicToken: resp.PublicToken,
				}
				updateTemplate.Execute(w, data)
			})
			log.Fatal(http.ListenAndServe(":9090", nil))
			return nil
		},
	}
}
