package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"gocount/internal/cgroups"
	"gocount/internal/container"

	"github.com/spf13/cobra"
)

var (
	flagMemory string
	flagCPU    string
)
var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command in a new container",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("GOCOUNT_CHILD") == "1" {
			childSetup(args)
			return
		}

		id := container.GenerateID()
		fmt.Println("Starting container:", id, "command:", args)

		// create cgroup before starting the child so we can configure limits
		cgPath, err := cgroups.Create(id)
		if err != nil {
			fmt.Println("Error creating cgroup:", err)
			os.Exit(1)
		}
		// set limits if provided (ignore errors but print)
		if err := cgroups.SetMemoryLimit(cgPath, flagMemory); err != nil {
			fmt.Println("Warning: cannot set memory limit:", err)
		}
		if err := cgroups.SetCPUQuota(cgPath, flagCPU); err != nil {
			fmt.Println("Warning: cannot set cpu quota:", err)
		}

		command := exec.Command("/proc/self/exe", append([]string{"run"}, args...)...)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Env = append(os.Environ(),
			"GOCOUNT_CHILD=1",
			"GOCOUNT_CONTAINER_ID="+id,
		)

		command.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUSER |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS,
			UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
			GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		}
		if err := container.EnsureContainerDir(); err != nil {
			fmt.Println("Error creating container dir:", err)
			os.Exit(1)
		}

		if err := command.Start(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		// Register in memory
		c := &container.Container{
			ID:      id,
			Pid:     command.Process.Pid,
			Command: args,
			Status:  "running",
			RootFs:  "./rootfs",
			Cgroup:  cgPath,
		}
		container.Containers[id] = c

		// Save to disk
		if err := container.SaveContainer(c); err != nil {
			fmt.Println("Error saving container:", err)
		}

		// Register container
		container.AddContainer(id, command.Process.Pid, args, "./rootfs")

		if err := command.Wait(); err != nil {
			fmt.Println("Error:", err)
		}
	},
}


var startCmd=&cobra.Command{
	Use:   "start [container_id]",
	Short: "Start an existing stopped container",
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

		fmt.Println("Starting container:", id, "command:", c.Command)
		
		// Fork a new process to run the container
		command := exec.Command("/proc/self/exe", append([]string{"run"}, c.Command...)...)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Env = append(os.Environ(),
			"GOCOUNT_CHILD=1",
			"GOCOUNT_CONTAINER_ID="+id,
		)

		command.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUSER |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS,
			UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
			GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		}

		if err := command.Start(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		// Update container info
		c.Pid = command.Process.Pid
		c.Status = "running"

		// Save updated status
		if err := container.SaveContainer(c); err != nil {
			fmt.Println("Error saving container:", err)
		}

		if err := command.Wait(); err != nil {
			fmt.Println("Error:", err)
		}
	},
}	



func childSetup(args []string) {
	// Add self to cgroup FIRST, before doing anything else
	containerID := os.Getenv("GOCOUNT_CONTAINER_ID")
	if containerID != "" {
		cgPath := filepath.Join("/sys/fs/cgroup", "gocount", containerID)
		pid := os.Getpid()
		if err := cgroups.AddProc(cgPath, pid); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot add self to cgroup: %v\n", err)
		}
	}

	syscall.Sethostname([]byte("gocount"))
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	syscall.Unmount("/proc", syscall.MNT_DETACH)
	syscall.Mount("proc", "/proc", "proc", 0, "")
	syscall.Exec(args[0], args, os.Environ())
}


func init() {
	rootCmd.AddCommand(runCmd)

	// Add flags
	runCmd.Flags().StringVar(&flagMemory, "memory", "", "Memory limit for container (e.g. 100M)")
	runCmd.Flags().StringVar(&flagCPU, "cpu", "", "CPU quota for container (cgroup v2 format: 'max' or '<quota> <period>')")

	rootCmd.AddCommand(startCmd)
}
