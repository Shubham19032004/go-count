package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
			containers, _ := container.LoadContainers()
			for _, cc := range containers {
				if cc.ID == id {
					c = cc
					break
				}
			}
		}

		fmt.Printf("Container Information:\n")
		fmt.Printf("  ID:        %s\n", c.ID)
		fmt.Printf("  Status:    %s\n", c.Status)
		fmt.Printf("  PID:       %d\n", c.Pid)
		fmt.Printf("  Command:   %v\n", c.Command)
		fmt.Printf("  RootFS:    %s\n", c.RootFs)
		fmt.Printf("  Cgroup:    %s\n", c.Cgroup)

		fmt.Printf("\nProcess Status:\n")
		if isProcessRunning(c.Pid) {
			fmt.Printf("  Running:   Yes\n")

			// Read process status from /proc
			if status := readProcStatus(c.Pid); status != "" {
				fmt.Printf("  State:     %s\n", status)
			}
		} else {
			fmt.Printf("  Running:   No\n")
		}

		// Cgroup resource limits and usage
		if c.Cgroup != "" {
			fmt.Printf("\nResource Limits:\n")
			showCgroupInfo(c.Cgroup)
		}

		// Namespaces
		fmt.Printf("\nNamespaces:\n")
		showNamespaces(c.Pid)
	},
}

func readProcStatus(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "State:") {
			return strings.TrimPrefix(line, "State:\t")
		}
	}
	return ""
}
func showCgroupInfo(cgPath string) {
	// Memory info
	if data, err := os.ReadFile(filepath.Join(cgPath, "memory.max")); err == nil {
		limit := strings.TrimSpace(string(data))
		if limit == "max" {
			fmt.Printf("  Memory Limit:    unlimited\n")
		} else {
			fmt.Printf("  Memory Limit:    %s bytes (%s)\n", limit, formatBytes(limit))
		}
	}

	if data, err := os.ReadFile(filepath.Join(cgPath, "memory.current")); err == nil {
		current := strings.TrimSpace(string(data))
		fmt.Printf("  Memory Usage:    %s bytes (%s)\n", current, formatBytes(current))
	}

	if data, err := os.ReadFile(filepath.Join(cgPath, "memory.peak")); err == nil {
		peak := strings.TrimSpace(string(data))
		fmt.Printf("  Memory Peak:     %s bytes (%s)\n", peak, formatBytes(peak))
	}

	// CPU info
	if data, err := os.ReadFile(filepath.Join(cgPath, "cpu.max")); err == nil {
		quota := strings.TrimSpace(string(data))
		if quota == "max" {
			fmt.Printf("  CPU Quota:       unlimited\n")
		} else {
			parts := strings.Fields(quota)
			if len(parts) == 2 {
				fmt.Printf("  CPU Quota:       %s/%s (%.1f%%)\n", parts[0], parts[1],
					float64(parseIntOrZero(parts[0]))/float64(parseIntOrZero(parts[1]))*100)
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(cgPath, "cpu.stat")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "usage_usec") {
				parts := strings.Fields(line)
				if len(parts) == 2 {
					usec := parseIntOrZero(parts[1])
					duration := time.Duration(usec) * time.Microsecond
					fmt.Printf("  CPU Time:        %s\n", duration)
				}
			}
		}
	}

	// Memory events (OOM kills, etc.)
	if data, err := os.ReadFile(filepath.Join(cgPath, "memory.events")); err == nil {
		fmt.Printf("\nMemory Events:\n")
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line != "" {
				parts := strings.Fields(line)
				if len(parts) == 2 && parts[1] != "0" {
					fmt.Printf("  %s: %s\n", parts[0], parts[1])
				}
			}
		}
	}

	// PIDs in cgroup
	if data, err := os.ReadFile(filepath.Join(cgPath, "cgroup.procs")); err == nil {
		pids := strings.TrimSpace(string(data))
		if pids != "" {
			pidList := strings.Split(pids, "\n")
			fmt.Printf("\nProcesses in Cgroup: %d\n", len(pidList))
			for _, pid := range pidList {
				fmt.Printf("  PID: %s\n", pid)
			}
		}
	}
}

func showNamespaces(pid int) {
	nsPath := fmt.Sprintf("/proc/%d/ns", pid)
	entries, err := os.ReadDir(nsPath)
	if err != nil {
		fmt.Printf("  Unable to read namespaces: %v\n", err)
		return
	}

	for _, entry := range entries {
		linkPath := filepath.Join(nsPath, entry.Name())
		if target, err := os.Readlink(linkPath); err == nil {
			fmt.Printf("  %s: %s\n", entry.Name(), target)
		}
	}
}

func formatBytes(bytesStr string) string {
	bytes := parseIntOrZero(bytesStr)
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func parseIntOrZero(s string) int64 {
	var val int64
	fmt.Sscanf(s, "%d", &val)
	return val
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Send signal 0 to check if process exists
	err = process.Signal(os.Signal(nil))
	return err == nil
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
