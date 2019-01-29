package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/freebot"
)

const (
	version = "0.1.0"
)

var (
	showVersion bool
	cfgFile     string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "", "c", "", "config file of freebot")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of freebot")
}

var rootCmd = &cobra.Command{
	Use:   "freebot",
	Short: "freebot is a github robot(https://github.com/fatedier/freebot)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version)
			return nil
		}

		content, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		cfg := freebot.Config{}
		err = json.Unmarshal(content, &cfg)
		if err != nil {
			fmt.Printf("parse config file error: %v\n", err)
			return nil
		}

		svc, err := freebot.NewService(cfg)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		svc.Run()

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
