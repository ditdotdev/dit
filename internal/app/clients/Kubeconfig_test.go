package clients

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFlattenKubeconfigInlinesFileReferences verifies the fix for issue #108:
// when the source kubeconfig references certificate-authority / client-certificate /
// client-key by file path, the flattened copy must inline them as *-data fields
// so the file references disappear. This is required because the datadatdat
// server container does not have access to the ~/.minikube/ directory holding
// the referenced certs, and on Windows the paths are host-absolute and
// non-resolvable inside a Linux container.
func TestFlattenKubeconfigInlinesFileReferences(t *testing.T) {
	tmp := t.TempDir()
	caPath := filepath.Join(tmp, "ca.crt")
	certPath := filepath.Join(tmp, "client.crt")
	keyPath := filepath.Join(tmp, "client.key")

	caData := []byte("-----BEGIN CERTIFICATE-----\nCA-PLACEHOLDER\n-----END CERTIFICATE-----\n")
	certData := []byte("-----BEGIN CERTIFICATE-----\nCLIENT-CERT-PLACEHOLDER\n-----END CERTIFICATE-----\n")
	keyData := []byte("-----BEGIN RSA PRIVATE KEY-----\nCLIENT-KEY-PLACEHOLDER\n-----END RSA PRIVATE KEY-----\n")

	for path, data := range map[string][]byte{caPath: caData, certPath: certData, keyPath: keyData} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	srcConfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: ` + caPath + `
    server: https://127.0.0.1:8443
  name: minikube
contexts:
- context:
    cluster: minikube
    user: minikube
  name: minikube
current-context: minikube
users:
- name: minikube
  user:
    client-certificate: ` + certPath + `
    client-key: ` + keyPath + `
`
	srcPath := filepath.Join(tmp, "config")
	if err := os.WriteFile(srcPath, []byte(srcConfig), 0600); err != nil {
		t.Fatalf("write src config: %v", err)
	}

	dstPath := filepath.Join(tmp, "nested", "config.flat")

	if err := FlattenKubeconfigToFile(srcPath, dstPath); err != nil {
		t.Fatalf("FlattenKubeconfigToFile: %v", err)
	}

	out, err := os.ReadFile(dstPath) // #nosec G304 -- dstPath is a path under t.TempDir() that this test just wrote
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	outStr := string(out)

	if strings.Contains(outStr, caPath) {
		t.Errorf("flattened config still contains CA file reference %q:\n%s", caPath, outStr)
	}
	if strings.Contains(outStr, certPath) {
		t.Errorf("flattened config still contains client cert file reference %q:\n%s", certPath, outStr)
	}
	if strings.Contains(outStr, keyPath) {
		t.Errorf("flattened config still contains client key file reference %q:\n%s", keyPath, outStr)
	}
	if !strings.Contains(outStr, "certificate-authority-data:") {
		t.Errorf("flattened config missing certificate-authority-data:\n%s", outStr)
	}
	if !strings.Contains(outStr, "client-certificate-data:") {
		t.Errorf("flattened config missing client-certificate-data:\n%s", outStr)
	}
	if !strings.Contains(outStr, "client-key-data:") {
		t.Errorf("flattened config missing client-key-data:\n%s", outStr)
	}
}

// TestFlattenKubeconfigPreservesInlineData verifies that kubeconfigs which
// already use inline *-data fields (e.g. Docker Desktop's built-in k8s) pass
// through unchanged — the function must not regress the working case.
func TestFlattenKubeconfigPreservesInlineData(t *testing.T) {
	tmp := t.TempDir()

	// Base64 of "ca-content", "cert-content", "key-content"
	srcConfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: Y2EtY29udGVudA==
    server: https://127.0.0.1:8443
  name: docker-desktop
contexts:
- context:
    cluster: docker-desktop
    user: docker-desktop
  name: docker-desktop
current-context: docker-desktop
users:
- name: docker-desktop
  user:
    client-certificate-data: Y2VydC1jb250ZW50
    client-key-data: a2V5LWNvbnRlbnQ=
`
	srcPath := filepath.Join(tmp, "config")
	if err := os.WriteFile(srcPath, []byte(srcConfig), 0600); err != nil {
		t.Fatalf("write src: %v", err)
	}
	dstPath := filepath.Join(tmp, "config.flat")

	if err := FlattenKubeconfigToFile(srcPath, dstPath); err != nil {
		t.Fatalf("FlattenKubeconfigToFile: %v", err)
	}

	out, err := os.ReadFile(dstPath) // #nosec G304 -- dstPath is a path under t.TempDir() that this test just wrote
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	outStr := string(out)

	if !strings.Contains(outStr, "Y2EtY29udGVudA==") {
		t.Errorf("flattened config dropped inline CA data:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Y2VydC1jb250ZW50") {
		t.Errorf("flattened config dropped inline client cert data:\n%s", outStr)
	}
	if !strings.Contains(outStr, "a2V5LWNvbnRlbnQ=") {
		t.Errorf("flattened config dropped inline client key data:\n%s", outStr)
	}
}

func TestFlattenKubeconfigMissingSourceReturnsError(t *testing.T) {
	tmp := t.TempDir()
	err := FlattenKubeconfigToFile(filepath.Join(tmp, "does-not-exist"), filepath.Join(tmp, "out"))
	if err == nil {
		t.Fatal("expected error for missing source kubeconfig, got nil")
	}
}
