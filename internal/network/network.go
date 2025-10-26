package network

import (
	"fmt"
	"os/exec"
)

// SetupVethPair creates a veth pair between host and container
func SetupVethPair(containerID string, pid int) error {
	hostIf := fmt.Sprintf("veth%s", containerID[:4])
	containerIf := "eth0"

	// 1. Create veth pair
	cmd := exec.Command("ip", "link", "add", hostIf, "type", "veth", "peer", "name", containerIf)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create veth pair: %v (%s)", err, string(out))
	}

	// 2. Move container end of veth into container's network namespace
	cmd = exec.Command("ip", "link", "set", containerIf, "netns", fmt.Sprintf("%d", pid))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to move veth to container: %v (%s)", err, string(out))
	}

	// 3. Bring up the host side
	exec.Command("ip", "link", "set", hostIf, "up").Run()

	// Optional: assign IP to host end (for testing)
	exec.Command("ip", "addr", "add", fmt.Sprintf("10.0.0.%d/24", pid%250), "dev", hostIf).Run()

	return nil
}
