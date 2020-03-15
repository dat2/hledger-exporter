package main

import (
	"fmt"
	"github.com/dat2/hledger-exporter/cmd"
	"os"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
