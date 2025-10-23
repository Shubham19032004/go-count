package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	CgroupRoot = "/sys/fs/cgroup"
	Prefix     = "gocount"
)

func EnsureCgroupRoot() error {
	rootPath := filepath.Join(CgroupRoot, Prefix)
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		if err := os.MkdirAll(rootPath, 0755); err != nil {
			return fmt.Errorf("cannot create cgroup root: %w", err)
		}
	}

	// Enable controllers in the parent cgroup
	subtreeControl := filepath.Join(rootPath, "cgroup.subtree_control")
	if err := writeFile(subtreeControl, "+cpu +memory +pids"); err != nil {
		return fmt.Errorf("cannot enable controllers: %w", err)
	}

	return nil
}

func Create(id string) (string, error) {
	if err := EnsureCgroupRoot(); err != nil {
		return "", err
	}
	path := filepath.Join(CgroupRoot, Prefix, id)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("mkdir cgroup: %w", err)
	}
	return path, nil
}

// SetMemoryLimit writes memory.max and enables OOM killing
func SetMemoryLimit(cgPath, limit string) error {
	if limit == "" {
		return nil // skip if not specified
	}

	// Convert "50M" to bytes if needed
	val := limit
	if strings.HasSuffix(limit, "M") {
		mbStr := strings.TrimSuffix(limit, "M")
		mb, err := strconv.ParseInt(mbStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid memory limit: %w", err)
		}
		val = strconv.FormatInt(mb*1024*1024, 10)
	} else if strings.HasSuffix(limit, "G") {
		gbStr := strings.TrimSuffix(limit, "G")
		gb, err := strconv.ParseInt(gbStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid memory limit: %w", err)
		}
		val = strconv.FormatInt(gb*1024*1024*1024, 10)
	}

	// Set memory limit
	if err := writeFile(filepath.Join(cgPath, "memory.max"), val); err != nil {
		return err
	}

	// Disable swap to enforce hard limit
	if err := writeFile(filepath.Join(cgPath, "memory.swap.max"), "0"); err != nil {
		// Non-fatal if swap control isn't available
		fmt.Printf("Warning: cannot disable swap: %v\n", err)
	}

	// Enable OOM group killing (kill all processes in cgroup on OOM)
	if err := writeFile(filepath.Join(cgPath, "memory.oom.group"), "1"); err != nil {
		// Non-fatal
		fmt.Printf("Warning: cannot enable oom.group: %v\n", err)
	}

	return nil
}

// SetCPUQuota writes cpu.max. `quota` example: "10000 100000" or "" for no limit.
func SetCPUQuota(cgPath, quota string) error {
	if quota == "" {
		return nil // skip if not specified
	}
	return writeFile(filepath.Join(cgPath, "cpu.max"), quota)
}

// AddProc writes pid into cgroup.procs to add a process
func AddProc(cgPath string, pid int) error {
	return writeFile(filepath.Join(cgPath, "cgroup.procs"), strconv.Itoa(pid))
}

// Delete removes the created cgroup directory (must be empty of procs)
func Delete(id string) error {
	path := filepath.Join(CgroupRoot, Prefix, id)
	return os.Remove(path)
}

// helper - don't use O_CREATE for cgroup files, they already exist
func writeFile(path, content string) error {
	// For cgroup files, don't use O_CREATE - they're created by the kernel
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
