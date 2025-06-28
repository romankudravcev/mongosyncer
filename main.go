package main

import (
	"log/slog"
	"os"

	"mongosyncer/pkg/api"
	"mongosyncer/pkg/config"
	"mongosyncer/pkg/downloader"
	"mongosyncer/pkg/mongosync"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	d := downloader.New(logger)
	apiClient := api.New(cfg.APIBaseURL, logger)
	syncManager := mongosync.New(cfg.BinaryPath, cfg.SourceURI, cfg.TargetURI, logger)

	if err := d.EnsureBinary(cfg.BinaryPath, cfg.DownloadURL); err != nil {
		logger.Error("Failed to ensure mongosync binary", "error", err)
		os.Exit(1)
	}

	if err := syncManager.Start(); err != nil {
		logger.Error("Failed to start mongosync", "error", err)
		os.Exit(1)
	}

	// Execute sync workflow
	if err := executeSyncWorkflow(apiClient, logger); err != nil {
		logger.Error("Sync workflow failed", "error", err)
		err := syncManager.Stop()
		if err != nil {
			logger.Error("Failed to stop mongosync process gracefully", "error", err)
		}
		os.Exit(1)
	}
}

func executeSyncWorkflow(apiClient *api.Client, logger *slog.Logger) error {
	// Start the sync process via API
	if err := apiClient.StartSync(); err != nil {
		return err
	}

	// Wait for sync to be ready for commit (monitoring progress every 5 seconds)
	if err := apiClient.WaitForCanCommit(); err != nil {
		return err
	}

	// Commit the sync
	if err := apiClient.Commit(); err != nil {
		return err
	}

	// Verify the sync is actually committed
	if err := apiClient.VerifyCommitted(); err != nil {
		return err
	}

	// Sync is complete and verified
	logger.Info("MongoDB synchronization completed and verified successfully!")
	return nil
}
