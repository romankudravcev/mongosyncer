package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client handles mongosync REST API interactions
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// CommitResponse represents the response from the commit endpoint
type CommitResponse struct {
	Success bool `json:"success"`
}

// ProgressResponse represents the response from the progress endpoint
type ProgressResponse struct {
	Progress struct {
		CanCommit bool   `json:"canCommit"`
		State     string `json:"state"`
		Info      string `json:"info"`
	} `json:"progress"`
	Success bool `json:"success"`
}

// StartRequest represents the request payload for starting sync
type StartRequest struct {
	Source                  string                 `json:"source"`
	Destination             string                 `json:"destination"`
	Verification            map[string]interface{} `json:"verification"`
	EnableUserWriteBlocking string                 `json:"enableUserWriteBlocking"`
}

// New creates a new API client
func New(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) WaitForReadyState() error {
	url := fmt.Sprintf("%s/progress", c.baseURL)
	for {
		resp, err := c.httpClient.Get(url)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var progress ProgressResponse
		if err := json.Unmarshal(body, &progress); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if progress.Progress.State != "IDLE" {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}

// StartSync initiates the synchronization process
func (c *Client) StartSync() error {
	url := fmt.Sprintf("%s/start", c.baseURL)

	payload := StartRequest{
		Source:      "cluster0",
		Destination: "cluster1",
		Verification: map[string]interface{}{
			"enabled": false,
		},
		EnableUserWriteBlocking: "sourceAndDestination",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal start request: %w", err)
	}

	c.logger.Info("Starting sync via REST API...")
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start sync failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Sync started successfully")
	return nil
}

// WaitForCanCommit polls the progress endpoint until canCommit is true
func (c *Client) WaitForCanCommit() error {
	url := fmt.Sprintf("%s/progress", c.baseURL)

	c.logger.Info("Monitoring sync progress...")

	for {
		resp, err := c.httpClient.Get(url)
		if err != nil {
			c.logger.Error("Failed to check progress", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			c.logger.Error("Failed to read progress response", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			c.logger.Warn("Progress check failed", "status", resp.StatusCode, "body", string(body))
			time.Sleep(5 * time.Second)
			continue
		}

		var progress ProgressResponse
		if err := json.Unmarshal(body, &progress); err != nil {
			c.logger.Error("Failed to parse progress response", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		c.logger.Info("Progress check", "state", progress.Progress.State, "canCommit", progress.Progress.CanCommit, "info", progress.Progress.Info)

		if progress.Progress.CanCommit {
			c.logger.Info("Sync is ready to commit!")
			return nil
		}

		c.logger.Info("Waiting for sync to be ready for commit...")
		time.Sleep(5 * time.Second)
	}
}

// Commit finalizes the synchronization process
func (c *Client) Commit() error {
	url := fmt.Sprintf("%s/commit", c.baseURL)

	c.logger.Info("Committing sync...")
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to commit sync: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read commit response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("commit failed with status %d: %s", resp.StatusCode, string(body))
	}

	var commitResp CommitResponse
	if err := json.Unmarshal(body, &commitResp); err != nil {
		return fmt.Errorf("failed to parse commit response: %w", err)
	}

	if !commitResp.Success {
		return fmt.Errorf("commit failed: success=false in response")
	}

	c.logger.Info("Sync committed successfully", "success", commitResp.Success)
	return nil
}

// VerifyCommitted checks if the sync state is COMMITTED, polling until it is
func (c *Client) VerifyCommitted() error {
	url := fmt.Sprintf("%s/progress", c.baseURL)

	c.logger.Info("Verifying sync is committed...")

	for {
		resp, err := c.httpClient.Get(url)
		if err != nil {
			c.logger.Error("Failed to verify commit status", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			c.logger.Error("Failed to read verification response", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			c.logger.Warn("Verification check failed", "status", resp.StatusCode, "body", string(body))
			time.Sleep(5 * time.Second)
			continue
		}

		var progress ProgressResponse
		if err := json.Unmarshal(body, &progress); err != nil {
			c.logger.Error("Failed to parse verification response", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		c.logger.Info("Verification check", "state", progress.Progress.State, "canCommit", progress.Progress.CanCommit, "info", progress.Progress.Info)

		if progress.Progress.State == "COMMITTED" {
			c.logger.Info("Sync verified as COMMITTED successfully!")
			return nil
		}

		c.logger.Info("Waiting for sync to reach COMMITTED state...", "currentState", progress.Progress.State)
		time.Sleep(5 * time.Second)
	}
}

// CleanDestination drops all user databases on the destination cluster
func (c *Client) CleanDestination(destinationURI string) error {
	c.logger.Info("Cleaning destination cluster...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to destination MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(destinationURI))
	if err != nil {
		return fmt.Errorf("failed to connect to destination cluster: %w", err)
	}
	defer client.Disconnect(ctx)

	// List all databases
	databases, err := client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	// Drop all user databases (skip system databases)
	systemDatabases := map[string]bool{
		"admin":  true,
		"local":  true,
		"config": true,
	}

	for _, dbName := range databases {
		if !systemDatabases[dbName] {
			c.logger.Info("Dropping database", "database", dbName)
			if err := client.Database(dbName).Drop(ctx); err != nil {
				return fmt.Errorf("failed to drop database %s: %w", dbName, err)
			}
		}
	}

	c.logger.Info("Destination cluster cleaned successfully")
	return nil
}

// ThoroughCleanDestination performs a more aggressive cleanup of the destination cluster
func (c *Client) ThoroughCleanDestination(destinationURI string) error {
	c.logger.Info("Performing thorough cleaning of destination cluster...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to destination MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(destinationURI))
	if err != nil {
		return fmt.Errorf("failed to connect to destination cluster: %w", err)
	}
	defer client.Disconnect(ctx)

	// First, drop all user databases
	databases, err := client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	systemDatabases := map[string]bool{
		"admin":  true,
		"local":  true,
		"config": true,
	}

	for _, dbName := range databases {
		if !systemDatabases[dbName] {
			c.logger.Info("Dropping user database", "database", dbName)
			if err := client.Database(dbName).Drop(ctx); err != nil {
				return fmt.Errorf("failed to drop database %s: %w", dbName, err)
			}
		}
	}

	// Clean potentially problematic collections in system databases
	// Clear the oplog in local database (this is often what causes the "existing data" error)
	localDB := client.Database("local")
	collections, err := localDB.ListCollectionNames(ctx, map[string]interface{}{})
	if err == nil {
		for _, collName := range collections {
			// Drop oplog and other replica set collections that might cause issues
			if collName == "oplog.rs" || collName == "replset.minvalid" || collName == "replset.oplogTruncateAfterPoint" {
				c.logger.Info("Dropping problematic collection", "database", "local", "collection", collName)
				if err := localDB.Collection(collName).Drop(ctx); err != nil {
					c.logger.Warn("Failed to drop collection", "database", "local", "collection", collName, "error", err)
				}
			}
		}
	}

	// Clear any user-created collections in admin database (but keep essential system ones)
	adminDB := client.Database("admin")
	adminCollections, err := adminDB.ListCollectionNames(ctx, map[string]interface{}{})
	if err == nil {
		essentialAdminCollections := map[string]bool{
			"system.users":   true,
			"system.roles":   true,
			"system.version": true,
		}

		for _, collName := range adminCollections {
			if !essentialAdminCollections[collName] && !strings.HasPrefix(collName, "system.") {
				c.logger.Info("Dropping non-essential admin collection", "collection", collName)
				if err := adminDB.Collection(collName).Drop(ctx); err != nil {
					c.logger.Warn("Failed to drop admin collection", "collection", collName, "error", err)
				}
			}
		}
	}

	c.logger.Info("Thorough destination cluster cleaning completed")
	return nil
}

// DiagnoseDestination provides detailed information about what exists on the destination cluster
func (c *Client) DiagnoseDestination(destinationURI string) error {
	c.logger.Info("Diagnosing destination cluster for existing data...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to destination MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(destinationURI))
	if err != nil {
		return fmt.Errorf("failed to connect to destination cluster: %w", err)
	}
	defer client.Disconnect(ctx)

	// List all databases
	databases, err := client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	c.logger.Info("Found databases", "count", len(databases), "databases", databases)

	// Check each database for collections and document counts
	for _, dbName := range databases {
		db := client.Database(dbName)
		collections, err := db.ListCollectionNames(ctx, map[string]interface{}{})
		if err != nil {
			c.logger.Warn("Failed to list collections", "database", dbName, "error", err)
			continue
		}

		c.logger.Info("Database details", "database", dbName, "collections", len(collections))

		for _, collName := range collections {
			coll := db.Collection(collName)
			count, err := coll.CountDocuments(ctx, map[string]interface{}{})
			if err != nil {
				c.logger.Warn("Failed to count documents", "database", dbName, "collection", collName, "error", err)
				continue
			}

			if count > 0 {
				c.logger.Info("Collection has documents", "database", dbName, "collection", collName, "count", count)
			}
		}
	}

	return nil
}
