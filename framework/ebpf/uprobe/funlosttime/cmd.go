package main

import (
	"funlosttime/monitor"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "sky_cloud_tool",
	Short:   "sky_cloud tool ",
	Long:    `provider ceph add record and check ceph config is right`,
	Version: "0.0.1",
}

func init() {
	rootCmd.AddCommand(monitor.FLTCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
