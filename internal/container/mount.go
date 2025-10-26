package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
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

	// Create essential device nodes in /dev
	if err := createDeviceNodes(); err != nil {
		return fmt.Errorf("failed to create device nodes: %v", err)
	}

	return nil
}

func createDeviceNodes() error {
	// Device nodes to create: name -> (type, major, minor, mode)
	devices := []struct {
		name  string
		mode  uint32
		major uint32
		minor uint32
	}{
		{"null", syscall.S_IFCHR | 0666, 1, 3},
		{"zero", syscall.S_IFCHR | 0666, 1, 5},
		{"full", syscall.S_IFCHR | 0666, 1, 7},
		{"random", syscall.S_IFCHR | 0666, 1, 8},
		{"urandom", syscall.S_IFCHR | 0666, 1, 9},
		{"tty", syscall.S_IFCHR | 0666, 5, 0},
		{"console", syscall.S_IFCHR | 0600, 5, 1},
	}

	for _, dev := range devices {
		path := filepath.Join("/dev", dev.name)
		devNum := int(unix.Mkdev(dev.major, dev.minor))

		if err := syscall.Mknod(path, dev.mode, devNum); err != nil {
			// Ignore if already exists
			if !os.IsExist(err) {
				return fmt.Errorf("mknod %s: %v", dev.name, err)
			}
		}
	}

	// Create /dev/pts directory for pseudo-terminals
	if err := os.MkdirAll("/dev/pts", 0755); err != nil {
		return fmt.Errorf("failed to create /dev/pts: %v", err)
	}

	// Create /dev/shm for shared memory
	if err := os.MkdirAll("/dev/shm", 0755); err != nil {
		return fmt.Errorf("failed to create /dev/shm: %v", err)
	}

	// Create standard file descriptors symlinks
	symlinks := map[string]string{
		"/dev/fd":     "/proc/self/fd",
		"/dev/stdin":  "/proc/self/fd/0",
		"/dev/stdout": "/proc/self/fd/1",
		"/dev/stderr": "/proc/self/fd/2",
	}

	for link, target := range symlinks {
		os.Remove(link) // Remove if exists
		if err := os.Symlink(target, link); err != nil {
			// Non-critical, just log
			fmt.Fprintf(os.Stderr, "Warning: failed to create symlink %s: %v\n", link, err)
		}
	}

	return nil
}
