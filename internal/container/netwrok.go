package container

import (
	"fmt"
	"os/exec"
)

// SetupNetworkInsideContainer configures eth0 and loopback
func SetupNetworkInsideContainer() error {
	// Bring up loopback
	if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
		return fmt.Errorf("failed to bring up loopback: %v", err)
	}

	// Bring up eth0 (container side of veth)
	if err := exec.Command("ip", "link", "set", "eth0", "up").Run(); err != nil {
		return fmt.Errorf("failed to bring up eth0: %v", err)
	}

	// Assign a static IP for now
	if err := exec.Command("ip", "addr", "add", "10.0.0.2/24", "dev", "eth0").Run(); err != nil {
		return fmt.Errorf("failed to assign IP: %v", err)
	}

	return nil
}
