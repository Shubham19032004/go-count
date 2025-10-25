package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func SetupMount(rootfs string) error {
	// Get absolute path
	rootfs, err := filepath.Abs(rootfs)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Make current mount propagation private (required for pivot_root)
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make / private: %v", err)
	}

	// Bind mount rootfs to itself (required before pivot_root)
	// CRITICAL: Third parameter MUST be empty string, not "bind"
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to bind mount rootfs: %v", err)
	}

	// Create directory for old root
	putold := filepath.Join(rootfs, ".pivot_root")
	if err := os.MkdirAll(putold, 0700); err != nil {
		return fmt.Errorf("failed to create putold: %v", err)
	}

	// Pivot root - switch to new root filesystem
	if err := syscall.PivotRoot(rootfs, putold); err != nil {
		return fmt.Errorf("pivot_root failed: %v", err)
	}

	// Change working directory to new root
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir failed: %v", err)
	}

	// Unmount old root
	if err := syscall.Unmount("/.pivot_root", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root failed: %v", err)
	}

	// Remove old root directory
	if err := os.RemoveAll("/.pivot_root"); err != nil {
		return fmt.Errorf("failed to remove old root: %v", err)
	}

	// Mount essential filesystems
	if err := os.MkdirAll("/proc", 0555); err != nil {
		return fmt.Errorf("failed to create /proc: %v", err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc failed: %v", err)
	}

	if err := os.MkdirAll("/sys", 0555); err != nil {
		return fmt.Errorf("failed to create /sys: %v", err)
	}
	if err := syscall.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		return fmt.Errorf("mount /sys failed: %v", err)
	}

	if err := os.MkdirAll("/dev", 0755); err != nil {
		return fmt.Errorf("failed to create /dev: %v", err)
	}
	if err := syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755"); err != nil {
		return fmt.Errorf("mount /dev failed: %v", err)
	}

	return nil
}