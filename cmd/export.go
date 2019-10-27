package cmd

import (
	"fmt"

	"github.com/araddon/dateparse"
	"github.com/spf13/cobra"
)

var dateTimeStr string

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&dateTimeStr, "start", "s", "", "Start date to print from")
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the transactions",
	Run: func(cmd *cobra.Command, args []string) {
		t, err := dateparse.ParseStrict(dateTimeStr)
		fmt.Println(t, err)
	},
}
