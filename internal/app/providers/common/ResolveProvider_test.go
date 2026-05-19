package common

import (
	"strings"
	"testing"

	"github.com/datadatdat/remote-sdk-go/remote"
	"github.com/stretchr/testify/assert"
)

// fakeRemote is a Remote stub used to exercise ResolveProvider without spawning real plugin subprocesses.
type fakeRemote struct {
	typ string
}

func (f *fakeRemote) Type() (string, error) { return f.typ, nil }
func (f *fakeRemote) FromURL(string, map[string]string) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeRemote) ToURL(map[string]interface{}) (string, map[string]string, error) {
	return "", nil, nil
}
func (f *fakeRemote) GetParameters(map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeRemote) ValidateRemote(map[string]interface{}) error     { return nil }
func (f *fakeRemote) ValidateParameters(map[string]interface{}) error { return nil }
func (f *fakeRemote) ListCommits(map[string]interface{}, map[string]interface{}, []remote.Tag) ([]remote.Commit, error) {
	return nil, nil
}
func (f *fakeRemote) GetCommit(map[string]interface{}, map[string]interface{}, string) (*remote.Commit, error) {
	return nil, nil
}

// TestResolveProviderUnknown verifies that asking for an unregistered remote produces a clean, named error.
// Pre-#46, callers did `remote.Get(name).Method(...)` and crashed with a nil-pointer dereference on a typo —
// this test pins the new "named error" behavior so regressions get caught immediately.
func TestResolveProviderUnknown(t *testing.T) {
	remote.ClearForTesting()

	_, err := ResolveProvider("not-a-real-provider")
	if !assert.Error(t, err) {
		return
	}
	assert.True(t, strings.Contains(err.Error(), "not-a-real-provider"),
		"error should name the missing provider, got: %s", err.Error())
}

// TestResolveProviderRegistered verifies the happy path — a registered provider is returned by name.
func TestResolveProviderRegistered(t *testing.T) {
	remote.ClearForTesting()
	defer remote.ClearForTesting()

	mock := &fakeRemote{typ: "resolveprovider-fake"}
	remote.Register(mock)

	got, err := ResolveProvider("resolveprovider-fake")
	assert.NoError(t, err)
	assert.Same(t, mock, got)
}
