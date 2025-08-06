package local

import (
	"context"
	"fmt"
	client "github.com/titan-data/titan-client-go"
	"os"
	"strings"
)

func init() {
	_, d := os.LookupEnv("TITAN_DEBUG")
	cfg.Debug = d
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var commitsApi = apiClient.CommitsApi
var repositoriesApi = apiClient.RepositoriesApi
var volumesApi = apiClient.VolumesApi
var ctx = context.Background()

func getKernel() string {
	var args = []string{"run", "--rm", "-i", "--privileged", "--pid=host", "alpine:latest",
		"nsenter", "-t", "1", "-m", "-u", "-n", "-i", "awk",
		"{ if ($1 == \"kernel:\") { inKernel = 1; next } if (inKernel == 1 && $1 == \"image:\") { print $2; inKernel = 0; quit } }",
		"/etc/linuxkit.yml"}
	v, err := ce.Exec("docker", args...)
	if err != nil {
		fmt.Println("Unable to locate kernel version")
		os.Exit(1)
	}
	return strings.TrimRight(v, "\n")
}

func getTag(k string) string {
	c := strings.Split(k, ":")[1]
	return strings.Split(c, "-")[0]
}

func zfsInstalled() bool {
	mod, _ := ce.Exec("docker", "run", "alpine:latest", "lsmod")
	for _, l := range strings.Split(mod, "\n") {
		for i, w := range strings.Split(l, " ") {
			if i == 0 && w == "zfs"{
				return true
			}
		}
	}
	return false
}

func installZFS(tag string) {
	fmt.Println("Installing ZFS for Docker Desktop")
	out, err := ce.Exec("docker", "run", "--privileged", "--rm", "titandata/docker-desktop-zfs-kernel:" + tag)
	if err != nil {
		if strings.Contains(out, "manifest unknown") {
			fmt.Println("Pre-built ZFS kernel modules not available for kernel version " + tag)
			fmt.Println("Falling back to building ZFS from source...")
			buildZFSFromSource(tag)
		} else {
			fmt.Println("Unable to install ZFS for Docker Desktop")
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func buildZFSFromSource(tag string) {
	fmt.Println("Building ZFS kernel modules from source (this may take 10-30 minutes)...")
	
	// Use the zfs-builder to compile modules for the current kernel
	buildArgs := []string{
		"run", "--rm", "--privileged",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-e", "ZFS_VERSION=zfs-0.8.2",
		"-e", "ZFS_CONFIG=kernel",
		"titandata/zfs-builder:latest",
	}
	
	out, err := ce.Exec("docker", buildArgs...)
	if err != nil {
		fmt.Println("Failed to build ZFS from source:")
		fmt.Println(out)
		fmt.Println("Error:", err)
		fmt.Println("")
		fmt.Println("You may need to:")
		fmt.Println("1. Ensure Docker Desktop is using a supported kernel version")
		fmt.Println("2. Try a different Docker Desktop version")
		fmt.Println("3. Install ZFS manually on your system")
		os.Exit(1)
	}
	
	fmt.Println("ZFS kernel modules built successfully")
}