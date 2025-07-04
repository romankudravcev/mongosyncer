package downloader

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// Downloader handles mongosync binary downloads
type Downloader struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Downloader {
	return &Downloader{
		logger: logger,
	}
}

// EnsureBinary downloads mongosync binary if it doesn't exist
func (d *Downloader) EnsureBinary(binaryPath, downloadURL string) error {
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		d.logger.Info("Mongosync binary not found, downloading...")
		return d.downloadBinary(binaryPath, downloadURL)
	}
	d.logger.Info("Mongosync binary already exists")
	return nil
}

func (d *Downloader) downloadBinary(binaryPath, downloadURL string) error {
	tmpTgz := "./mongosync.tgz"

	d.logger.Info("Downloading mongosync binary...")
	curlCmd := exec.Command("curl", "-L", "-o", tmpTgz, downloadURL)
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr
	if err := curlCmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	d.logger.Info("Extracting mongosync binary...")
	// Use simple extraction approach like the working version
	extractCmd := exec.Command("tar", "-xzf", tmpTgz, "--strip-components=2", "mongosync-ubuntu2404-x86_64-1.14.0/bin/mongosync")
	if err := extractCmd.Run(); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// If the binary path is not "./mongosync", rename it
	if binaryPath != "./mongosync" {
		if err := os.Rename("./mongosync", binaryPath); err != nil {
			return fmt.Errorf("failed to rename binary: %w", err)
		}
	}

	// Make the binary executable
	d.logger.Info("Making mongosync binary executable...")
	chmodCmd := exec.Command("chmod", "+x", binaryPath)
	if err := chmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	if err := os.Remove(tmpTgz); err != nil {
		d.logger.Warn("Failed to remove temp archive", "error", err)
	}

	d.logger.Info("Mongosync binary downloaded successfully", "path", binaryPath)
	return nil
}
