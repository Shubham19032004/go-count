package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "os"
)

var rootCmd = &cobra.Command{
    Use:   "gocount",
    Short: "gocount is a minimal container runtime",
    Long:  `Run Linux processes in isolated namespaces, like a tiny Docker.`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
