// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetProviderState clears viper config + the Providers map between tests
// so cases stay independent.
func resetProviderState(t *testing.T) {
	t.Helper()
	origProviders := Providers
	origContexts := viper.GetStringMap("contexts")
	t.Cleanup(func() {
		Providers = origProviders
		viper.Set("contexts", origContexts)
	})
	Providers = make(map[string]Provider)
	viper.Set("contexts", map[string]interface{}{})
}

// usingTempViperFile points viper at a temp dir for WriteConfig calls.
func usingTempViperFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("contexts:\n"), 0600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	viper.SetConfigFile(cfg)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read temp config: %v", err)
	}
	return cfg
}

func TestLoadAndWriteContext_RoundTrip(t *testing.T) {
	in := context{isDefault: true, host: "localhost", port: 5001, contextType: ProviderTypeDocker}
	m := writeContext(in)
	out := loadContext(m)
	if out != in {
		t.Errorf("round trip: got %+v want %+v", out, in)
	}
}

func TestCreate_Docker(t *testing.T) {
	resetProviderState(t)
	p := Create("my-ctx", ProviderTypeDocker, 1234)
	if p == nil {
		t.Fatal("Create returned nil")
	}
	if p.GetName() != "my-ctx" || p.GetPort() != 1234 || p.GetType() != "docker" {
		t.Errorf("docker provider wrong: name=%s port=%d type=%s", p.GetName(), p.GetPort(), p.GetType())
	}
}

func TestCreate_Kubernetes(t *testing.T) {
	resetProviderState(t)
	p := Create("k8s-ctx", ProviderTypeKubernetes, 5678)
	if p == nil {
		t.Fatal("Create returned nil")
	}
	if p.GetType() != "kubernetes" {
		t.Errorf("expected kubernetes type, got %s", p.GetType())
	}
}

// Regression: an unknown type used to fall through the switch and return a
// nil Provider, which the install command then panicked on. It must exit
// with a clear error instead.
func TestCreate_UnknownType_Exits(t *testing.T) {
	resetProviderState(t)
	didExit, code := captureExit(t, func() {
		Create("weird", "unknown-type", 1234)
	})
	if !didExit || code != 1 {
		t.Errorf("expected exit code 1 for unknown provider type, got didExit=%v code=%d", didExit, code)
	}
}

// Regression: SetDefault with an unknown name used to un-default every
// context and default none, leaving the config with no default context.
func TestSetDefault_UnknownContext_ExitsAndKeepsDefault(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t) // AddProvider writes config; keep it off ~/.dit/config
	p := Create("real-ctx", ProviderTypeDocker, 1234)
	AddProvider(p)

	didExit, code := captureExit(t, func() {
		SetDefault("not-a-context")
	})
	if !didExit || code != 1 {
		t.Errorf("expected exit code 1 for unknown context, got didExit=%v code=%d", didExit, code)
	}
	if got := DefaultName(); got != "real-ctx" {
		t.Errorf("existing default must survive a failed SetDefault; DefaultName() = %q, want %q", got, "real-ctx")
	}
}

func TestList_ReturnsProvidersMap(t *testing.T) {
	resetProviderState(t)
	p := Local("ctx", "localhost", 1)
	Providers["ctx"] = p
	out := List()
	if out["ctx"] != p {
		t.Errorf("List did not return the providers map")
	}
}

func TestGetAvailablePort_ReturnsNonZero(t *testing.T) {
	port := GetAvailablePort()
	if port <= 0 {
		t.Errorf("GetAvailablePort returned %d", port)
	}
}

func TestAddProvider_PersistsToViper(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t)
	p := Local("ctx-a", "localhost", 5001)
	AddProvider(p)

	contexts := viper.GetStringMap("contexts")
	if _, ok := contexts["ctx-a"]; !ok {
		t.Errorf("AddProvider did not write context to viper")
	}
}

func TestAddProvider_FirstIsDefault(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t)
	AddProvider(Local("first", "localhost", 5001))

	contexts := viper.GetStringMap("contexts")
	c := loadContext(contexts["first"])
	if !c.isDefault {
		t.Errorf("first provider should be marked default")
	}

	AddProvider(Local("second", "localhost", 5002))
	contexts = viper.GetStringMap("contexts")
	c2 := loadContext(contexts["second"])
	if c2.isDefault {
		t.Errorf("second provider should not be default")
	}
}

func TestDefaultName_SingleContext(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{
		"only": writeContext(context{isDefault: false, host: "localhost", port: 1, contextType: "docker"}),
	})
	if got := DefaultName(); got != "only" {
		t.Errorf("DefaultName single ctx = %q, want only", got)
	}
}

func TestDefaultName_MultipleWithDefault(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{
		"a": writeContext(context{isDefault: false, host: "localhost", port: 1, contextType: "docker"}),
		"b": writeContext(context{isDefault: true, host: "localhost", port: 2, contextType: "docker"}),
	})
	if got := DefaultName(); got != "b" {
		t.Errorf("DefaultName = %q, want b", got)
	}
}

func TestDefaultName_NoContextsPanics(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DefaultName with no contexts should panic")
		}
	}()
	_ = DefaultName()
}

func TestDefaultName_MultipleWithoutDefault(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{
		"a": writeContext(context{isDefault: false, host: "localhost", port: 1, contextType: "docker"}),
		"b": writeContext(context{isDefault: false, host: "localhost", port: 2, contextType: "docker"}),
	})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("DefaultName with multiple ctxs and no default should panic")
		}
	}()
	_ = DefaultName()
}

func TestDefault_ReturnsProviderForDefaultName(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{
		"a": writeContext(context{isDefault: true, host: "localhost", port: 1, contextType: "docker"}),
	})
	p := Local("a", "localhost", 1)
	Providers["a"] = p
	if Default() != p {
		t.Errorf("Default did not return mapped provider")
	}
}

func TestRemove_DropsContext(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t)
	AddProvider(Local("ctx-a", "localhost", 5001))
	AddProvider(Local("ctx-b", "localhost", 5002))

	Remove("ctx-a")
	contexts := viper.GetStringMap("contexts")
	if _, ok := contexts["ctx-a"]; ok {
		t.Errorf("Remove did not drop ctx-a")
	}
	if _, ok := contexts["ctx-b"]; !ok {
		t.Errorf("Remove dropped wrong context")
	}
}

func TestRemove_PromotesNewDefault(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t)
	AddProvider(Local("ctx-a", "localhost", 5001)) // becomes default
	AddProvider(Local("ctx-b", "localhost", 5002))

	Remove("ctx-a") // delete the default — ctx-b should be promoted

	contexts := viper.GetStringMap("contexts")
	c := loadContext(contexts["ctx-b"])
	if !c.isDefault {
		t.Errorf("Remove should have promoted ctx-b to default")
	}
}

func TestRemove_LastContext_NoPromotion(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t)
	AddProvider(Local("only", "localhost", 5001))
	Remove("only")
	contexts := viper.GetStringMap("contexts")
	if len(contexts) != 0 {
		t.Errorf("expected empty contexts after removing only one, got %d", len(contexts))
	}
}

func TestSetDefault_SwitchesDefault(t *testing.T) {
	resetProviderState(t)
	usingTempViperFile(t)
	AddProvider(Local("ctx-a", "localhost", 5001))
	AddProvider(Local("ctx-b", "localhost", 5002))

	SetDefault("ctx-b")
	contexts := viper.GetStringMap("contexts")
	a := loadContext(contexts["ctx-a"])
	b := loadContext(contexts["ctx-b"])
	if a.isDefault {
		t.Errorf("ctx-a should no longer be default")
	}
	if !b.isDefault {
		t.Errorf("ctx-b should be the new default")
	}
}

func TestByName_QualifiedName(t *testing.T) {
	resetProviderState(t)
	p := Local("ctx", "localhost", 1)
	Providers["ctx"] = p
	got, name := ByName("ctx/myrepo")
	if got != p || name != "myrepo" {
		t.Errorf("ByName(ctx/myrepo) = (%v, %q), want (%v, myrepo)", got, name, p)
	}
}

func TestByName_UnqualifiedFallsToDefault(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{
		"only": writeContext(context{isDefault: false, host: "localhost", port: 1, contextType: "docker"}),
	})
	p := Local("only", "localhost", 1)
	Providers["only"] = p
	got, name := ByName("repo")
	if got != p || name != "repo" {
		t.Errorf("ByName(repo) = (%v, %q), want (%v, repo)", got, name, p)
	}
}

func TestByName_UnknownContextPanics(t *testing.T) {
	resetProviderState(t)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("ByName for unknown ctx should panic")
		}
	}()
	_, _ = ByName("nope/repo")
}

func TestResolve_QualifiedName(t *testing.T) {
	resetProviderState(t)
	p := Local("ctx", "localhost", 1)
	Providers["ctx"] = p
	got, name := Resolve("", "ctx/myrepo")
	if got != p || name != "myrepo" {
		t.Errorf("Resolve(ctx/myrepo) = (%v, %q)", got, name)
	}
}

func TestResolve_ContextFlag(t *testing.T) {
	resetProviderState(t)
	p := Local("ctx", "localhost", 1)
	Providers["ctx"] = p
	got, name := Resolve("ctx", "myrepo")
	if got != p || name != "myrepo" {
		t.Errorf("Resolve(--context=ctx, myrepo) = (%v, %q)", got, name)
	}
}

func TestResolve_DefaultFallback(t *testing.T) {
	resetProviderState(t)
	viper.Set("contexts", map[string]interface{}{
		"only": writeContext(context{isDefault: false, host: "localhost", port: 1, contextType: "docker"}),
	})
	p := Local("only", "localhost", 1)
	Providers["only"] = p
	got, name := Resolve("", "myrepo")
	if got != p || name != "myrepo" {
		t.Errorf("Resolve(default) = (%v, %q)", got, name)
	}
}

func TestProviderTypeConstants(t *testing.T) {
	if ProviderTypeDocker != "docker" {
		t.Errorf("ProviderTypeDocker = %q", ProviderTypeDocker)
	}
	if ProviderTypeKubernetes != "kubernetes" {
		t.Errorf("ProviderTypeKubernetes = %q", ProviderTypeKubernetes)
	}
}

// captureExit branches in ProviderFactory.

func TestResolve_UnknownQualifiedContext_Exits(t *testing.T) {
	resetProviderState(t)
	didExit, code := captureExit(t, func() { Resolve("", "nope/repo") })
	if !didExit || code != 1 {
		t.Errorf("expected osExit(1), got didExit=%v code=%d", didExit, code)
	}
}

func TestResolve_UnknownContextFlag_Exits(t *testing.T) {
	resetProviderState(t)
	didExit, code := captureExit(t, func() { Resolve("nope", "repo") })
	if !didExit || code != 1 {
		t.Errorf("expected osExit(1), got didExit=%v code=%d", didExit, code)
	}
}

func TestCreate_AlreadyExists_Exits(t *testing.T) {
	resetProviderState(t)
	Providers["dup"] = Local("dup", "h", 1)
	didExit, code := captureExit(t, func() {
		_ = Create("dup", ProviderTypeDocker, 1)
	})
	if !didExit || code != 1 {
		t.Errorf("expected osExit(1), got didExit=%v code=%d", didExit, code)
	}
}
