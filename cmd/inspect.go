package cmd

import (
    "fmt"

    "gocount/internal/container"

    "github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
    Use:   "inspect [container_id]",
    Short: "Inspect a container",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        id := args[0]
        c, ok := container.Containers[id]
        if !ok {
            fmt.Println("Container not found:", id)
            return
        }

        fmt.Printf("ID: %s\nPID: %d\nStatus: %s\nCommand: %v\nRootFS: %s\n",
            c.ID, c.Pid, c.Status, c.Command, c.RootFs)
    },
}

func init() {
    rootCmd.AddCommand(inspectCmd)
}
