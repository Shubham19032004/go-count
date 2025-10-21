package cmd

import (
    "fmt"

    "gocount/internal/container"
    "github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
    Use:   "ps",
    Short: "List all running containers",
    Run: func(cmd *cobra.Command, args []string) {
        if err := container.EnsureContainerDir(); err != nil {
            fmt.Println("Error:", err)
            return
        }

        containers, err := container.LoadContainers()
        if err != nil {
            fmt.Println("Error loading containers:", err)
            return
        }

        fmt.Println("CONTAINER ID\tPID\tSTATUS\tCOMMAND")
        for _, c := range containers {
            fmt.Printf("%s\t%d\t%s\t%v\n", c.ID, c.Pid, c.Status, c.Command)
        }
    },
}



func init() {
    rootCmd.AddCommand(psCmd)
}
