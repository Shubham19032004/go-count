package rootfs

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	// Alpine Linux minirootfs - small and reliable
	DefaultRootfsURL = "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz"

)

// EnsureRootfs checks if rootfs exists and is valid, downloads if needed
func EnsureRootfs(RootfsDir string) error {
	// Check if rootfs exists and has basic directories
	if isValidRootfs(RootfsDir) {
		return nil
	}

	fmt.Println("Rootfs not found or invalid. Downloading Alpine Linux rootfs...")
	return DownloadAndExtractRootfs(DefaultRootfsURL, RootfsDir)
}

// isValidRootfs checks if the rootfs directory contains a valid filesystem
func isValidRootfs(rootfsPath string) bool {
	// Check for essential directories that should exist in any Linux rootfs
	essentialDirs := []string{"bin", "lib", "etc", "usr"}

	for _, dir := range essentialDirs {
		path := filepath.Join(rootfsPath, dir)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return false
		}
	}

	// Check if /bin/sh exists (most basic requirement)
	sh := filepath.Join(rootfsPath, "bin", "sh")
	if _, err := os.Stat(sh); err != nil {
		return false
	}

	return true
}

// DownloadAndExtractRootfs downloads and extracts a rootfs tarball
func DownloadAndExtractRootfs(url, destPath string) error {
	// Create rootfs directory if it doesn't exist
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs directory: %v", err)
	}

	// Download the tarball
	fmt.Printf("Downloading from %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download rootfs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download rootfs: HTTP %d", resp.StatusCode)
	}

	// Create gzip reader
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	// Extract tarball
	fmt.Println("Extracting rootfs...")
	if err := extractTar(gzr, destPath); err != nil {
		return fmt.Errorf("failed to extract rootfs: %v", err)
	}

	fmt.Println("Rootfs setup complete!")
	return nil
}

// extractTar extracts a tar archive to the destination path
func extractTar(r io.Reader, destPath string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destPath, header.Name)

		// Ensure the parent directory exists
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()

		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil && !os.IsExist(err) {
				return err
			}

		case tar.TypeLink:
			linkTarget := filepath.Join(destPath, header.Linkname)
			if err := os.Link(linkTarget, target); err != nil && !os.IsExist(err) {
				return err
			}
		}
	}

	return nil
}

// GetRootfsPath returns the path to the rootfs directory
func GetRootfsPath(RootfsDir string) string {
	return RootfsDir
}
