package cmd

import (
	"fmt"
	"os"
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
		err := syscall.Kill(c.Pid, syscall.SIGKILL)
		if err != nil {
			fmt.Println("Error in stop container:", err)
		}
		c.Status = "stopped"

		// Save updated status
		container.SaveContainer(c)
		fmt.Println("Container stopped:", id)
	},
}
var removeCmd = &cobra.Command{
	Use:   "rm [container_id]",
	Short: "remove container",
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
		if c.Pid > 0 {
			if err := syscall.Kill(c.Pid, syscall.SIGKILL); err != nil {
				fmt.Printf("Warning: failed to kill process %d: %v\n", c.Pid, err)
			} else {
				fmt.Printf("Container process %d killed\n", c.Pid)
			}
		}
		path := fmt.Sprintf("/tmp/gocount/%s.json", c.ID)
		if err := os.Remove(path); err != nil {
			fmt.Printf("Warning: failed to remove metadata file: %v\n", err)
		}

		// Step 5: Remove from in-memory map
		delete(container.Containers, c.ID)

		fmt.Println("Container", c.ID, "removed successfully.")

	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(removeCmd)
}
