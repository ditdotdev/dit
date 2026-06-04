package clients

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// FlattenKubeconfigToFile reads a kubeconfig at srcPath, inlines any
// certificate-authority / client-certificate / client-key file references
// as their base64-encoded *-data counterparts, and writes the result to
// dstPath (creating parent directories as needed).
//
// The dit server container only mounts ~/.kube/ from the host and
// has no access to the ~/.minikube/ (or equivalent) directory holding the
// referenced cert files. On Windows, those references are host-absolute
// paths that a Linux container cannot resolve regardless. Flattening makes
// the kubeconfig self-contained so the container can consume it without
// needing the referenced files on disk.
func FlattenKubeconfigToFile(srcPath, dstPath string) error {
	cfg, err := clientcmd.LoadFromFile(srcPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", srcPath, err)
	}
	if err := clientcmdapi.FlattenConfig(cfg); err != nil {
		return fmt.Errorf("flatten: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dstPath), err)
	}
	if err := clientcmd.WriteToFile(*cfg, dstPath); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	return nil
}
