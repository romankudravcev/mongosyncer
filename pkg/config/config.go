package config

import (
	"errors"
	"os"
)

type Config struct {
	SourceURI   string
	TargetURI   string
	BinaryPath  string
	APIBaseURL  string
	DownloadURL string
}

func LoadConfig() (*Config, error) {
	sourceURI := os.Getenv("MONGOSYNC_SOURCE")
	targetURI := os.Getenv("MONGOSYNC_TARGET")

	if sourceURI == "" || targetURI == "" {
		return nil, errors.New("MONGOSYNC_SOURCE and MONGOSYNC_TARGET environment variables must be set")
	}

	return &Config{
		SourceURI:   sourceURI,
		TargetURI:   targetURI,
		BinaryPath:  "./mongosync",
		APIBaseURL:  "http://localhost:27182/api/v1",
		DownloadURL: "https://fastdl.mongodb.org/tools/mongosync/mongosync-ubuntu2404-x86_64-1.14.0.tgz",
	}, nil
}
