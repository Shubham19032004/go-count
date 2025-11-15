package network

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SetupVethPair creates a veth pair between host and container
func SetupVethPair(containerID string, pid int) error {
	// Use unique names for both ends initially
	hostIf := fmt.Sprintf("veth-%s", containerID[:8])
	containerIfTemp := fmt.Sprintf("vethc-%s", containerID[:7]) // Temporary name
	containerIf := "eth0"                                       // Final name inside container

	// Cleanup any existing interfaces with the same names
	CleanupVeth(hostIf)
	CleanupVeth(containerIfTemp)

	// 1. Create veth pair with unique names
	cmd := exec.Command("ip", "link", "add", hostIf, "type", "veth", "peer", "name", containerIfTemp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create veth pair: %v (%s)", err, string(out))
	}

	// 2. Move container end of veth into container's network namespace
	cmd = exec.Command("ip", "link", "set", containerIfTemp, "netns", fmt.Sprintf("%d", pid))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to move veth to container: %v (%s)", err, string(out))
	}

	// 3. Rename the interface to eth0 inside the container namespace
	cmd = exec.Command("nsenter", "-t", fmt.Sprintf("%d", pid), "-n", "ip", "link", "set", containerIfTemp, "name", containerIf)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename veth in container: %v (%s)", err, string(out))
	}

	// 4. Bring up the host side
	if err := exec.Command("ip", "link", "set", hostIf, "up").Run(); err != nil {
		return fmt.Errorf("failed to bring up host veth: %v", err)
	}

	// 5. Assign IP to host end
	cmd = exec.Command("ip", "addr", "add", "10.0.0.1/24", "dev", hostIf)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore error if address already exists
		if !strings.Contains(string(out), "exists") {
			return fmt.Errorf("failed to assign host IP: %v (%s)", err, string(out))
		}
	}

	// 6. Enable IP forwarding
	if err := EnableIPForwarding(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to enable IP forwarding: %v\n", err)
	}

	// 7. Setup NAT for internet access
	if err := SetupNAT(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to setup NAT: %v\n", err)
	}

	return nil
}

// EnableIPForwarding enables IPv4 forwarding on the host
func EnableIPForwarding() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %v (%s)", err, string(out))
	}
	return nil
}

// SetupNAT configures iptables for container internet access
func SetupNAT() error {
	// Check if rule already exists
	cmd := exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING", "-s", "10.0.0.0/24", "-j", "MASQUERADE")
	if err := cmd.Run(); err != nil {
		// Rule doesn't exist, add it
		cmd = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "10.0.0.0/24", "-j", "MASQUERADE")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to add NAT rule: %v (%s)", err, string(out))
		}
	}

	// Allow forwarding for veth interfaces - INSERT at top to bypass Docker rules
	cmd = exec.Command("iptables", "-C", "FORWARD", "-i", "veth-+", "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		// Use INSERT (-I) instead of APPEND (-A) to put rule at the top
		cmd = exec.Command("iptables", "-I", "FORWARD", "1", "-i", "veth-+", "-j", "ACCEPT")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to insert FORWARD rule: %v (%s)\n", err, string(out))
		}
	}

	cmd = exec.Command("iptables", "-C", "FORWARD", "-o", "veth-+", "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		// Use INSERT (-I) instead of APPEND (-A) to put rule at the top
		cmd = exec.Command("iptables", "-I", "FORWARD", "1", "-o", "veth-+", "-j", "ACCEPT")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to insert FORWARD rule: %v (%s)\n", err, string(out))
		}
	}

	// Also allow established connections back
	cmd = exec.Command("iptables", "-C", "FORWARD", "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("iptables", "-I", "FORWARD", "1", "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT")
		cmd.Run() // Ignore error
	}

	return nil
}

// CleanupVeth removes a veth interface if it exists
func CleanupVeth(ifName string) error {
	// Check if interface exists
	cmd := exec.Command("ip", "link", "show", ifName)
	if err := cmd.Run(); err != nil {
		// Interface doesn't exist, nothing to cleanup
		return nil
	}

	// Delete the interface
	cmd = exec.Command("ip", "link", "delete", ifName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete veth %s: %v", ifName, err)
	}

	return nil
}

// CleanupContainerNetwork removes network resources for a container
func CleanupContainerNetwork(containerID string) {
	hostIf := fmt.Sprintf("veth-%s", containerID[:8])
	CleanupVeth(hostIf)
}
// SetupNetworkInsideContainer configures eth0 and loopback inside container
func SetupNetworkInsideContainer() error {
	fmt.Println("DEBUG: Starting network setup inside container...")

	// Bring up loopback
	fmt.Println("DEBUG: Bringing up loopback...")
	if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
		return fmt.Errorf("failed to bring up loopback: %v", err)
	}

	// Check if eth0 exists
	fmt.Println("DEBUG: Checking for eth0...")
	cmd := exec.Command("ip", "link", "show", "eth0")
	if err := cmd.Run(); err != nil {
		// eth0 doesn't exist - this should not happen at this point
		return fmt.Errorf("eth0 not found: %v", err)
	}
	fmt.Println("DEBUG: eth0 found!")

	// Bring up eth0 (container side of veth)
	fmt.Println("DEBUG: Bringing up eth0...")
	if err := exec.Command("ip", "link", "set", "eth0", "up").Run(); err != nil {
		return fmt.Errorf("failed to bring up eth0: %v", err)
	}

	// Small delay to let the interface come up
	time.Sleep(50 * time.Millisecond)

	// Assign a static IP
	fmt.Println("DEBUG: Assigning IP to eth0...")
	cmd = exec.Command("ip", "addr", "add", "10.0.0.2/24", "dev", "eth0")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore if address already assigned
		if !strings.Contains(string(out), "exists") {
			return fmt.Errorf("failed to assign IP: %v (%s)", err, string(out))
		}
		fmt.Println("DEBUG: IP already assigned")
	} else {
		fmt.Println("DEBUG: IP assigned successfully")
	}

	// Verify IP was assigned
	cmd = exec.Command("ip", "addr", "show", "eth0")
	if out, err := cmd.CombinedOutput(); err == nil {
		fmt.Printf("DEBUG: eth0 configuration:\n%s\n", string(out))
	}

	// Check current routes before adding
	fmt.Println("DEBUG: Routes before adding default:")
	cmd = exec.Command("ip", "route", "show")
	if out, err := cmd.CombinedOutput(); err == nil {
		fmt.Println(string(out))
	}

	// Add default route - CRITICAL for internet access
	// First, check if default route already exists
	cmd = exec.Command("ip", "route", "show", "default")
	out, _ := cmd.CombinedOutput()
	
	if len(out) > 0 {
		fmt.Printf("DEBUG: Default route already exists: %s\n", string(out))
		// Delete existing default route first
		fmt.Println("DEBUG: Deleting existing default route...")
		exec.Command("ip", "route", "del", "default").Run()
	}

	fmt.Println("DEBUG: Adding default route via 10.0.0.1...")
	cmd = exec.Command("ip", "route", "add", "default", "via", "10.0.0.1")
	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := string(out)
		fmt.Printf("ERROR: Failed to add default route: %v\nOutput: %s\n", err, outStr)
		
		// Try alternative method
		fmt.Println("DEBUG: Trying alternative route add method...")
		cmd = exec.Command("ip", "route", "add", "default", "via", "10.0.0.1", "dev", "eth0")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to add default route (both methods): %v (%s)", err, string(out))
		}
	}
	fmt.Println("DEBUG: Default route added successfully")

	// VERIFY the route was actually added
	fmt.Println("DEBUG: Verifying routes after adding default:")
	cmd = exec.Command("ip", "route", "show")
	if out, err := cmd.CombinedOutput(); err == nil {
		routeOutput := string(out)
		fmt.Println(routeOutput)
		
		// Check if default route is present
		if !strings.Contains(routeOutput, "default") {
			return fmt.Errorf("CRITICAL: Default route was not added successfully!")
		}
	}

	// Test connectivity to gateway
	fmt.Println("DEBUG: Testing connectivity to gateway (10.0.0.1)...")
	cmd = exec.Command("ping", "-c", "1", "-W", "2", "10.0.0.1")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("WARNING: Cannot ping gateway: %v\n%s\n", err, string(out))
	} else {
		fmt.Println("DEBUG: Gateway reachable!")
	}

	// Test connectivity to internet
	fmt.Println("DEBUG: Testing connectivity to internet (8.8.8.8)...")
	cmd = exec.Command("ping", "-c", "1", "-W", "2", "8.8.8.8")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("WARNING: Cannot ping internet: %v\n%s\n", err, string(out))
	} else {
		fmt.Println("DEBUG: Internet reachable!")
	}

	// Verify DNS resolution
	fmt.Println("DEBUG: Testing DNS resolution...")
	cmd = exec.Command("nslookup", "google.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: DNS resolution test failed: %v\n%s\n", err, string(out))
		// Check resolv.conf
		content, _ := os.ReadFile("/etc/resolv.conf")
		fmt.Printf("DEBUG: /etc/resolv.conf contents:\n%s\n", string(content))
	} else {
		fmt.Println("DEBUG: DNS resolution working!")
	}

	fmt.Println("DEBUG: Network setup complete!")
	return nil
}