package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"gocount/internal/container"

	"github.com/spf13/cobra"
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

		command := exec.Command("/proc/self/exe", append([]string{"run"}, args...)...)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Env = append(os.Environ(), "GOCOUNT_CHILD=1")

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

func childSetup(args []string) {
	syscall.Sethostname([]byte("gocount"))
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	syscall.Unmount("/proc", syscall.MNT_DETACH)
	syscall.Mount("proc", "/proc", "proc", 0, "")
	syscall.Exec(args[0], args, os.Environ())
}

func init() {
	rootCmd.AddCommand(runCmd)
}
