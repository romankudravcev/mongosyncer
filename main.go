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
		fmt.Println("Downloading mongosync binary with curl...")
		tmpTgz := "./mongosync.tgz"
		curlCmd := exec.Command("curl", "-L", "-o", tmpTgz, "https://fastdl.mongodb.org/tools/mongosync/mongosync-ubuntu2404-x86_64-1.14.0.tgz")
		curlCmd.Stdout = os.Stdout
		curlCmd.Stderr = os.Stderr
		err := curlCmd.Run()
		if err != nil {
			fmt.Println("curl download failed:", err)
			return
		}
		fmt.Println("Extracting mongosync binary...")
		err = exec.Command("tar", "-xzf", tmpTgz).Run()
		if err != nil {
			fmt.Println("Extraction failed:", err)
			return
		}
		err = os.Remove(tmpTgz)
		if err != nil {
			fmt.Println("Failed to remove temp archive:", err)
			return
		}
		fmt.Println("Listing current directory contents:")
		lsCmd := exec.Command("ls", "-l", "./")
		lsCmd.Stdout = os.Stdout
		lsCmd.Stderr = os.Stderr
		_ = lsCmd.Run()
	}

	sourceURI := os.Getenv("MONGOSYNC_SOURCE")
	targetURI := os.Getenv("MONGOSYNC_TARGET")
	if sourceURI == "" || targetURI == "" {
		fmt.Println("Please set MONGOSYNC_SOURCE and MONGOSYNC_TARGET environment variables.")
		return
	}

	fmt.Println("Running mongosync...")
	cmd := exec.Command(binPath, "--source", sourceURI, "--target", targetURI)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Println("mongosync failed:", err)
		return
	}
	fmt.Println("mongosync finished successfully.")
}
