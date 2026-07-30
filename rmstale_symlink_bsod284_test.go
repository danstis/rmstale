//go:build !windows
// +build !windows

// Symlink-creating tests for BSOD-284 live behind !windows because
// os.Symlink requires elevated privilege or Developer Mode on Windows.
// The production guard in rmstale.go uses only os.Lstat / os.ModeSymlink,
// which are platform-neutral stdlib, so the Windows release build is
// unaffected. Every symlink target here is a disposable t.TempDir()
// populated with sentinel files; a guard regression can therefore only
// delete test fixtures, never real data.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestValidateNoSymlink exercises the symlink guard added for BSOD-284.
// Real directories must pass in both modes; symlinks must be refused
// without --follow-symlinks and accepted with it; and non-existent paths
// must not be masked (deferred to procDir's os.Stat error handling).
func TestValidateNoSymlink(t *testing.T) {
	realDir := t.TempDir()

	// Build a disposable symlink target inside t.TempDir() so even a
	// total guard failure cannot damage anything but throwaway fixtures.
	targetDir := t.TempDir()
	sentinel := filepath.Join(targetDir, "sentinel")
	if err := os.WriteFile(sentinel, []byte("do not delete"), 0o600); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}
	linkParent := t.TempDir()
	linkPath := filepath.Join(linkParent, "link-to-target")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	// Non-existent path: stays a no-op so procDir surfaces the error.
	missingDir := filepath.Join(t.TempDir(), "does-not-exist")

	tests := []struct {
		name           string
		path           string
		followSymlinks bool
		wantErr        bool
		wantInMessage  string
	}{
		{"real dir refused-by-default", realDir, false, false, ""},
		{"real dir with opt-in", realDir, true, false, ""},
		{"symlink refused by default", linkPath, false, true, "symbolic link"},
		{"symlink allowed with --follow-symlinks", linkPath, true, false, ""},
		{"missing path returned as nil", missingDir, false, false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNoSymlink(tc.path, tc.followSymlinks)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateNoSymlink(%q, %v) error = %v, wantErr = %v",
					tc.path, tc.followSymlinks, err, tc.wantErr)
			}
			if tc.wantInMessage != "" && err != nil && !strings.Contains(err.Error(), tc.wantInMessage) {
				t.Fatalf("validateNoSymlink(%q, %v) error %q should contain %q",
					tc.path, tc.followSymlinks, err.Error(), tc.wantInMessage)
			}
		})
	}
}

// TestUsageDocumentsFollowSymlinks confirms the help text surfaces the new
// flag so a future default-flip cannot pass CI without an updated usage().
func TestUsageDocumentsFollowSymlinks(t *testing.T) {
	if !strings.Contains(usage(), "--follow-symlinks") {
		t.Fatalf("usage() must mention --follow-symlinks; got:\n%s", usage())
	}
}

// TestMainRejectsSymlinkPath is the integration regression test for
// BSOD-284: a --path that resolves to a symbolic link must not cause
// main() to descend into procDir. The subprocess exits
// exitProcessingError (= 1) when the guard fires; exitSuccess would mean
// the guard let the symlink through. The sentinel file behind the
// symlink must still exist after the subprocess exits, so even a total
// guard failure can only delete disposable test fixtures.
func TestMainRejectsSymlinkPath(t *testing.T) {
	if os.Getenv("BE_CRASHER_SYMLINK") == "1" {
		runCrasherSymlink(t)
		return
	}

	// Disposable target: a t.TempDir() containing a sentinel stale file.
	// A symlink at /tmp-style-path points at it. The guard must refuse
	// before procDir can touch the sentinel.
	targetDir := t.TempDir()
	sentinelPath := filepath.Join(targetDir, "must-survive")
	if err := os.WriteFile(sentinelPath, []byte("untouched"), 0o600); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}
	linkParent := t.TempDir()
	linkPath := filepath.Join(linkParent, "rmstale-symlink-path")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	cmd := exec.Command(testBinaryPath, "-test.run=TestMainRejectsSymlinkPath")
	cmd.Env = append(os.Environ(),
		"BE_CRASHER_SYMLINK=1",
		fmt.Sprintf("RMSTALE_TEST_PATH=%s", linkPath),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit code, got nil. output: %s", out)
	}
	if !strings.Contains(string(out), "symbolic link") {
		t.Fatalf("expected 'symbolic link' error message, got %q (err=%v)", out, err)
	}
	if !exists(sentinelPath) {
		t.Fatalf("sentinel behind the symlink was deleted; guard did not fire in time")
	}
}

// TestMainAllowsSymlinkPathWithFlag is the positive control for BSOD-284:
// when the caller opts in with --follow-symlinks, rmstale must descend
// into the symlinked directory and remove the stale sentinel file. This
// proves the override path still works for callers who genuinely need
// the old follow-the-symlink behaviour.
func TestMainAllowsSymlinkPathWithFlag(t *testing.T) {
	if os.Getenv("BE_CRASHER_SYMLINK_ALLOW") == "1" {
		runCrasherSymlinkAllow(t)
		return
	}

	targetDir := t.TempDir()
	sentinelPath := filepath.Join(targetDir, "stale-sentinel")
	if err := os.WriteFile(sentinelPath, []byte("will be removed"), 0o600); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}
	// Force the sentinel into the past so isStale treats it as stale.
	past := mustPast(t)
	if err := os.Chtimes(sentinelPath, past, past); err != nil {
		t.Fatalf("os.Chtimes sentinel: %v", err)
	}

	linkParent := t.TempDir()
	linkPath := filepath.Join(linkParent, "rmstale-symlink-allow")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	cmd := exec.Command(testBinaryPath, "-test.run=TestMainAllowsSymlinkPathWithFlag")
	cmd.Env = append(os.Environ(),
		"BE_CRASHER_SYMLINK_ALLOW=1",
		fmt.Sprintf("RMSTALE_TEST_PATH=%s", linkPath),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 with --follow-symlinks, got err=%v output=%q", err, out)
	}
	if exists(sentinelPath) {
		t.Fatalf("sentinel behind the symlink was not processed; --follow-symlinks path failed")
	}
}

// runCrasherSymlink is the BE_CRASHER subprocess for the rejection test.
// It invokes run() with -p <symlink> and no --follow-symlinks. The guard
// must fire; if it does not, procDir will follow the symlink and remove
// the sentinel. Either way the subprocess exits with run()'s return code
// so the parent test can distinguish the two outcomes.
func runCrasherSymlink(t *testing.T) {
	path := os.Getenv("RMSTALE_TEST_PATH")
	if path == "" {
		t.Fatal("crasher requires RMSTALE_TEST_PATH")
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// -a 1 makes everything stale so procDir (if reached) would delete
	// the sentinel. -y skips the interactive prompt. --follow-symlinks
	// is intentionally omitted so the guard fires.
	os.Args = []string{"rmstale", "-a", "1", "-p", path, "-y"}
	os.Exit(run())
}

// runCrasherSymlinkAllow is the BE_CRASHER subprocess for the positive
// control. It passes --follow-symlinks so the tool must descend through
// the symlink and remove the sentinel.
func runCrasherSymlinkAllow(t *testing.T) {
	path := os.Getenv("RMSTALE_TEST_PATH")
	if path == "" {
		t.Fatal("crasher requires RMSTALE_TEST_PATH")
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale", "-a", "1", "-p", path, "-y", "--follow-symlinks"}
	os.Exit(run())
}

// mustPast returns a timestamp safely in the past so the sentinel is
// considered stale by isStale's age comparison. Imported here to avoid
// relying on the suite's setAge helper which requires *testing.T.
func mustPast(t *testing.T) (past time.Time) {
	t.Helper()
	now := time.Now()
	return now.AddDate(-1, 0, 0)
}
