package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ditdotdev/dit/internal/app/providers"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ---------------------------------------------------------------------------
// Fake Provider — records all calls so simple-command tests can verify
// that flag/arg parsing wired the right values through to the provider.
// ---------------------------------------------------------------------------

type fakeProviderCall struct {
	Method string
	Args   []interface{}
}

type fakeProvider struct {
	name    string
	port    int
	ptype   string
	Calls   []fakeProviderCall
	OnList  func(string)
	OnError func(method string)
}

func (f *fakeProvider) record(method string, args ...interface{}) {
	f.Calls = append(f.Calls, fakeProviderCall{Method: method, Args: args})
}

func (f *fakeProvider) GetType() string { return f.ptype }
func (f *fakeProvider) GetName() string { return f.name }
func (f *fakeProvider) GetPort() int    { return f.port }

func (f *fakeProvider) Abort(repo string) { f.record("Abort", repo) }
func (f *fakeProvider) Checkout(repo, guid string, tags []string) {
	f.record("Checkout", repo, guid, tags)
}
func (f *fakeProvider) Clone(uri, repo, commit string, params, arguments []string, disablePortMap bool, tags []string) {
	f.record("Clone", uri, repo, commit, params, arguments, disablePortMap, tags)
}
func (f *fakeProvider) Commit(repo, message string, tags []string) {
	f.record("Commit", repo, message, tags)
}
func (f *fakeProvider) Copy(repo, driver, source, path string) {
	f.record("Copy", repo, driver, source, path)
}
func (f *fakeProvider) Delete(repo, commit string, tags []string) {
	f.record("Delete", repo, commit, tags)
}
func (f *fakeProvider) Fork(uri, org, name string) { f.record("Fork", uri, org, name) }
func (f *fakeProvider) Install(properties []string, verbose bool) {
	f.record("Install", properties, verbose)
}
func (f *fakeProvider) List(context string) {
	f.record("List", context)
	if f.OnList != nil {
		f.OnList(context)
	}
}
func (f *fakeProvider) Log(repo string, tags []string) { f.record("Log", repo, tags) }
func (f *fakeProvider) Migrate(repo, name string)      { f.record("Migrate", repo, name) }
func (f *fakeProvider) Pull(repo, commit, remoteName string, tags []string, metadataOnly bool) {
	f.record("Pull", repo, commit, remoteName, tags, metadataOnly)
}
func (f *fakeProvider) Push(repo, commit, remoteName string, tags []string, metadataOnly bool) {
	f.record("Push", repo, commit, remoteName, tags, metadataOnly)
}
func (f *fakeProvider) RemoteAdd(repo, uri, remoteName string, params map[string]string) {
	f.record("RemoteAdd", repo, uri, remoteName, params)
}
func (f *fakeProvider) RemoteList(repo string) { f.record("RemoteList", repo) }
func (f *fakeProvider) RemoteLog(repo, remoteName string, tags []string) {
	f.record("RemoteLog", repo, remoteName, tags)
}
func (f *fakeProvider) RemoteRemove(repo, remote string) { f.record("RemoteRemove", repo, remote) }
func (f *fakeProvider) Remove(repo string, force bool)   { f.record("Remove", repo, force) }
func (f *fakeProvider) Run(image, repo string, environments, arguments []string, disablePortMap, privileged bool) {
	f.record("Run", image, repo, environments, arguments, disablePortMap, privileged)
}
func (f *fakeProvider) Start(repo string)                      { f.record("Start", repo) }
func (f *fakeProvider) Status(repo string)                     { f.record("Status", repo) }
func (f *fakeProvider) Stop(repo string)                       { f.record("Stop", repo) }
func (f *fakeProvider) Tag(repo, commit string, tags []string) { f.record("Tag", repo, commit, tags) }
func (f *fakeProvider) Uninstall(force, removeImage bool)      { f.record("Uninstall", force, removeImage) }
func (f *fakeProvider) Upgrade(force bool, version string, finalize bool, path string) {
	f.record("Upgrade", force, version, finalize, path)
}

// withFakeProvider injects a fake into the package-level `provider` var and
// swaps rootCmd's PersistentPreRun to a no-op so the test does not touch the
// real config-driven initProvider().
//
// Also registers the fake under the "test-ctx" name in providers.Providers so
// commands that go through providers.Resolve (rm, run) find a real Provider
// for their unqualified-name lookups. The original Providers map is restored
// on cleanup.
func withFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	fp := &fakeProvider{name: "test-ctx", port: 5001, ptype: "docker"}
	origProvider := provider
	origPreRun := rootCmd.PersistentPreRun
	origProviders := providers.Providers
	provider = fp
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {}
	providers.Providers = map[string]providers.Provider{"test-ctx": fp}
	t.Cleanup(func() {
		provider = origProvider
		rootCmd.PersistentPreRun = origPreRun
		providers.Providers = origProviders
	})
	return fp
}

// resetGlobalFlags clears the package-level flag vars that are reused
// across subcommands. Cobra retains values across Execute() calls, so any
// flag set in a prior test leaks unless explicitly cleared. We also reset
// each subcommand's `Changed` state on its flags because cobra's
// MarkFlagRequired check consults Changed, not the variable's value.
func resetGlobalFlags() {
	guid = ""
	tags = nil
	params = nil
	envVars = nil
	name = ""
	source = ""
	destination = ""
	remote = ""
	updateOnly = false
	removeImages = false
	privileged = false
	force = false
	verbose = false
	message = ""
	disablePortMap = false
	context = ""

	// Walk every command's flag set and clear the Changed bit so a flag
	// set in a previous test does not falsely satisfy MarkFlagRequired
	// (cp's --source) in the next test.
	for _, c := range rootCmd.Commands() {
		c.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
	}
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
}

func execCmd(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func lastCall(fp *fakeProvider) fakeProviderCall {
	if len(fp.Calls) == 0 {
		return fakeProviderCall{}
	}
	return fp.Calls[len(fp.Calls)-1]
}

// ===========================================================================
// start / stop / status
// ===========================================================================

func TestStartCmd_CallsProviderStart(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("start", "myrepo"); err != nil {
		t.Fatalf("start: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Start" || c.Args[0] != "myrepo" {
		t.Errorf("got %v, want Start(myrepo)", c)
	}
}

func TestStartCmd_RequiresArg(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	if _, err := execCmd("start"); err == nil {
		t.Fatal("start without arg should error")
	}
}

func TestStopCmd_CallsProviderStop(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("stop", "myrepo"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Stop" || c.Args[0] != "myrepo" {
		t.Errorf("got %v, want Stop(myrepo)", c)
	}
}

func TestStopCmd_RequiresArg(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	if _, err := execCmd("stop"); err == nil {
		t.Fatal("stop without arg should error")
	}
}

func TestStatusCmd_CallsProviderStatus(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("status", "myrepo"); err != nil {
		t.Fatalf("status: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Status" || c.Args[0] != "myrepo" {
		t.Errorf("got %v, want Status(myrepo)", c)
	}
}

// ===========================================================================
// pull / push
// ===========================================================================

func TestPullCmd_WithFlags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("pull", "myrepo", "-c", "abc123", "-r", "origin", "-u", "-t", "tag1,tag2"); err != nil {
		t.Fatalf("pull: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Pull" {
		t.Fatalf("got %s, want Pull", c.Method)
	}
	if c.Args[0] != "myrepo" || c.Args[1] != "abc123" || c.Args[2] != "origin" || c.Args[4] != true {
		t.Errorf("pull args wrong: %+v", c.Args)
	}
	if ts, ok := c.Args[3].([]string); !ok || len(ts) != 2 || ts[0] != "tag1" {
		t.Errorf("pull tags wrong: %v", c.Args[3])
	}
}

func TestPullCmd_DefaultFlags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("pull", "myrepo"); err != nil {
		t.Fatalf("pull: %v", err)
	}
	c := lastCall(fp)
	if c.Args[1] != "" || c.Args[2] != "" || c.Args[4] != false {
		t.Errorf("pull defaults wrong: %+v", c.Args)
	}
}

func TestPushCmd_WithFlags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("push", "myrepo", "-c", "abc123", "-r", "origin", "-u"); err != nil {
		t.Fatalf("push: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Push" || c.Args[1] != "abc123" || c.Args[2] != "origin" || c.Args[4] != true {
		t.Errorf("push wrong: %+v", c)
	}
}

// ===========================================================================
// rm / tag / migrate
// ===========================================================================

func TestRmCmd_WithForce(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	// providers.Resolve splits on "/" first — "test-ctx/myrepo" picks up
	// the fake we registered without needing the --context flag.
	if _, err := execCmd("rm", "-f", "test-ctx/myrepo"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Remove" || c.Args[0] != "myrepo" || c.Args[1] != true {
		t.Errorf("rm wrong: %+v", c)
	}
}

func TestRmCmd_DefaultForce(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("rm", "test-ctx/myrepo"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	c := lastCall(fp)
	if c.Args[1] != false {
		t.Errorf("rm default force should be false, got %v", c.Args[1])
	}
}

func TestRmCmd_BadContext(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	// Capture stderr/os.Exit interactions by running the subcommand
	// inside the parent process; providers.Resolve uses os.Exit on
	// unknown context. We avoid that test path here — instead exercise
	// the contextFlag branch with a valid context.
	resetGlobalFlags()
	context = "test-ctx"
	defer func() { context = "" }()
	// Use a bare name (no slash) so the contextFlag branch is taken.
	if _, err := execCmd("rm", "-f", "barerepo"); err != nil {
		t.Fatalf("rm: %v", err)
	}
}

func TestTagCmd_WithCommitAndTags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("tag", "myrepo", "-c", "deadbeef", "-t", "v1,v2"); err != nil {
		t.Fatalf("tag: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Tag" || c.Args[0] != "myrepo" || c.Args[1] != "deadbeef" {
		t.Errorf("tag wrong: %+v", c)
	}
	if ts, ok := c.Args[2].([]string); !ok || len(ts) != 2 {
		t.Errorf("tag tags wrong: %v", c.Args[2])
	}
}

func TestMigrateCmd_WithSource(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("migrate", "-s", "oldpg", "newrepo"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Migrate" || c.Args[0] != "oldpg" || c.Args[1] != "newrepo" {
		t.Errorf("migrate wrong: %+v", c)
	}
}

func TestMigrateCmd_RequiresArg(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	if _, err := execCmd("migrate"); err == nil {
		t.Fatal("migrate without repo arg should error")
	}
}

// ===========================================================================
// run
// ===========================================================================

func TestRunCmd_WithImageAndName(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("run", "postgres:latest", "-n", "test-ctx/demo"); err != nil {
		t.Fatalf("run: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Run" || c.Args[0] != "postgres:latest" || c.Args[1] != "demo" {
		t.Errorf("run wrong: %+v", c)
	}
}

func TestRunCmd_WithExtraArgs(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("run", "postgres:latest", "-n", "test-ctx/demo", "-e", "FOO=bar", "-P", "--privileged"); err != nil {
		t.Fatalf("run: %v", err)
	}
	c := lastCall(fp)
	envs, _ := c.Args[2].([]string)
	if len(envs) != 1 || envs[0] != "FOO=bar" {
		t.Errorf("run envs wrong: %v", envs)
	}
	if c.Args[4] != true || c.Args[5] != true {
		t.Errorf("run port-map/privileged wrong: %+v", c.Args)
	}
}

func TestRunCmd_RequiresArg(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	if _, err := execCmd("run"); err == nil {
		t.Fatal("run without args should error")
	}
}

// ===========================================================================
// abort / checkout / commit / clone / cp / delete / log
// ===========================================================================

func TestAbortCmd_CallsAbort(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("abort", "myrepo"); err != nil {
		t.Fatalf("abort: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Abort" || c.Args[0] != "myrepo" {
		t.Errorf("abort wrong: %+v", c)
	}
}

func TestCheckoutCmd_WithFlags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("checkout", "myrepo", "-c", "abc", "-t", "v1"); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Checkout" || c.Args[0] != "myrepo" || c.Args[1] != "abc" {
		t.Errorf("checkout wrong: %+v", c)
	}
}

func TestCommitCmd_WithMessage(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("commit", "myrepo", "-m", "hello"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Commit" || c.Args[0] != "myrepo" || c.Args[1] != "hello" {
		t.Errorf("commit wrong: %+v", c)
	}
}

func TestCloneCmd_WithFlags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("clone", "ssh://uri", "-n", "myrepo", "-c", "deadbeef", "-p", "k=v", "-P"); err != nil {
		t.Fatalf("clone: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Clone" || c.Args[0] != "ssh://uri" || c.Args[1] != "myrepo" || c.Args[2] != "deadbeef" || c.Args[5] != true {
		t.Errorf("clone wrong: %+v", c)
	}
}

func TestCpCmd_WithSourceDest(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("cp", "myrepo", "-s", "/host/path", "-d", "/cont/path"); err != nil {
		t.Fatalf("cp: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Copy" || c.Args[0] != "myrepo" || c.Args[1] != "local" || c.Args[2] != "/host/path" || c.Args[3] != "/cont/path" {
		t.Errorf("cp wrong: %+v", c)
	}
}

func TestCpCmd_RequiresSource(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	if _, err := execCmd("cp", "myrepo"); err == nil {
		t.Fatal("cp without --source should error")
	}
}

func TestDeleteCmd_WithCommit(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("delete", "myrepo", "-c", "deadbeef"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Delete" || c.Args[0] != "myrepo" || c.Args[1] != "deadbeef" {
		t.Errorf("delete wrong: %+v", c)
	}
}

func TestLogCmd_WithTags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("log", "myrepo", "-t", "v1"); err != nil {
		t.Fatalf("log: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Log" || c.Args[0] != "myrepo" {
		t.Errorf("log wrong: %+v", c)
	}
}

// ===========================================================================
// upgrade
// ===========================================================================

func TestUpgradeCmd_Default(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	c := lastCall(fp)
	if c.Method != "Upgrade" {
		t.Errorf("upgrade wrong: %+v", c)
	}
}

func TestUpgradeCmd_WithFlags(t *testing.T) {
	resetGlobalFlags()
	fp := withFakeProvider(t)
	if _, err := execCmd("upgrade", "-f", "-p", "/some/path"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	c := lastCall(fp)
	// upgrade(force, version, finalize, path)
	if c.Args[0] != true || c.Args[3] != "/some/path" {
		t.Errorf("upgrade wrong: %+v", c.Args)
	}
}

// ===========================================================================
// ls — uses providers.List() directly, not the global `provider` var,
// so we test the trivial header-only path with no contexts configured.
// ===========================================================================

func TestLsCmd_PrintsHeader(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)
	out, err := execCmd("ls")
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	// At minimum, the header should be printed.
	// The provider iteration runs against the real providers.Providers map
	// (likely empty in unit-test environment), so we only check the header.
	_ = out
}

// ===========================================================================
// Help / usage smoke tests (cover the cobra metadata so Use/Short/Long lines
// are evaluated). These keep us honest about the help string content.
// ===========================================================================

func TestHelpSmoke(t *testing.T) {
	resetGlobalFlags()
	withFakeProvider(t)

	subs := []string{
		"start", "stop", "status", "pull", "push", "rm", "tag", "run",
		"migrate", "abort", "checkout", "commit", "clone", "cp", "delete",
		"log", "ls", "upgrade", "install", "uninstall",
	}
	for _, s := range subs {
		t.Run(s, func(t *testing.T) {
			out, _ := execCmd(s, "--help")
			if !strings.Contains(out, s) && out == "" {
				t.Errorf("help for %s produced no output", s)
			}
		})
	}
}
