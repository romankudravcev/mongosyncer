package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

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
		err = exec.Command("tar", "-xzf", tmpTgz, "--strip-components=2", "mongosync-ubuntu2404-x86_64-1.14.0/bin/mongosync").Run()
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
	cmd := exec.Command(binPath, "--acceptRemoteAPIRequest", "--disableVerification", "--acceptDisclaimer", "--cluster0", sourceURI, "--cluster1", targetURI)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		fmt.Println("mongosync failed to start:", err)
		return
	}

	// Wait a few seconds for mongosync to initialize
	time.Sleep(5 * time.Second)

	fmt.Println("Triggering sync start via REST API...")
	curlCmd := exec.Command("curl", "-XPOST", "http://localhost:27182/api/v1/start", "-H", "Content-Type: application/json", "--data", `{"source":"cluster0","destination":"cluster1"}`)
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr
	_ = curlCmd.Run()

	err = cmd.Wait()
	if err != nil {
		fmt.Println("mongosync failed:", err)
		return
	}
	fmt.Println("mongosync finished successfully.")
}
