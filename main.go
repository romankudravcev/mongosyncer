package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

func downloadMongosync(dest string) error {
	url := "https://fastdl.mongodb.org/tools/mongosync/mongosync-ubuntu2404-x86_64-1.14.0.tgz"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	binPath := "./mongosync"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		fmt.Println("Downloading mongosync binary...")
		tmpTgz := "./mongosync.tgz"
		err := downloadMongosync(tmpTgz)
		if err != nil {
			fmt.Println("Download failed:", err)
			return
		}
		// Extract the binary (assume tar.gz contains 'mongosync' at top level)
		err = exec.Command("tar", "-xzf", tmpTgz).Run()
		if err != nil {
			return
		}
		err = os.Remove(tmpTgz)
		if err != nil {
			return
		}
	}

	sourceURI := os.Getenv("MONGOSYNC_SOURCE")
	targetURI := os.Getenv("MONGOSYNC_TARGET")
	if sourceURI == "" || targetURI == "" {
		fmt.Println("Please set MONGOSYNC_SOURCE and MONGOSYNC_TARGET environment variables.")
		return
	}

	cmd := exec.Command(binPath, "--source", sourceURI, "--target", targetURI)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Println("Running mongosync...")
	if err := cmd.Run(); err != nil {
		fmt.Println("mongosync failed:", err)
	}
}
