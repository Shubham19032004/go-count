package cmd

import (
	"fmt"
	"syscall"

	"gocount/internal/container"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [container_id]",
	Short: "Stop a running container",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		c, ok := container.Containers[id]
		if !ok {
			// Load from disk if not in memory
			containers, _ := container.LoadContainers()
			for _, cc := range containers {
				if cc.ID == id {
					c = cc
					break
				}
			}
		}

		if c == nil {
			fmt.Println("Container not found:", id)
			return
		}

		// Kill the container
		syscall.Kill(c.Pid, syscall.SIGKILL)
		c.Status = "stopped"

		// Save updated status
		container.SaveContainer(c)
		fmt.Println("Container stopped:", id)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
