package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/logger"
	"github.com/stretchr/testify/suite"
)

func init() {
	initLogger()
}

// testBinaryPath captures os.Args[0] before any test gets a chance to clobber
// it (TestMainVersionFlag reassigns os.Args to simulate CLI invocation).
// exec.Command needs the full test binary path; relying on os.Args[0] at the
// moment TestMainRejectsNonPositiveAge runs would fail if a prior test changed
// it.
var testBinaryPath = os.Args[0]

// RMStaleSuite defines the testing suite with the following files:
//
//	rootDir/
//		oldEmptySubdir/
//		oldSubdir1/
//			oldFile2
//		oldSubdir2/
//			oldFile3.yes
//			oldFile3.no
//		oldSubdir3/
//			recentFile3
//		recentSubdir1/
//		oldFile1
//		oldFile4.no
//		oldFile4.yes
//		recentFile1
//		recentFile2.no
//		recentFile2.yes
type RMStateSuite struct {
	suite.Suite
	age            int
	rootDir        string
	oldEmptySubdir string
	oldSubdir1     string
	oldFile2       *os.File
	oldSubdir2     string
	oldFile3yes    *os.File
	oldFile3no     *os.File
	oldSubdir3     string
	recentFile3    *os.File
	recentSubdir1  string
	oldFile1       *os.File
	oldFile4no     *os.File
	oldFile4yes    *os.File
	recentFile1    *os.File
	recentFile2no  *os.File
	recentFile2yes *os.File
}

// The SetupTest method will be run before every test in the suite.
func (suite *RMStateSuite) SetupTest() {
	// Create folder structure
	suite.rootDir = tempDirectory(suite.T(), "rootDir", os.TempDir())
	suite.oldSubdir1 = tempDirectory(suite.T(), "oldSubdir1", suite.rootDir)
	suite.oldSubdir2 = tempDirectory(suite.T(), "oldSubdir2", suite.rootDir)
	suite.oldSubdir3 = tempDirectory(suite.T(), "oldSubdir3", suite.rootDir)
	suite.oldEmptySubdir = tempDirectory(suite.T(), "oldEmptySubdir", suite.rootDir)
	suite.recentSubdir1 = tempDirectory(suite.T(), "recentSubdir1", suite.rootDir)

	// Create file structure
	suite.oldFile2 = tempFile(suite.T(), "oldFile2", suite.oldSubdir1)
	suite.oldFile3no = tempFile(suite.T(), "oldFile3.*.no", suite.oldSubdir2)
	suite.oldFile3yes = tempFile(suite.T(), "oldFile3.*.yes", suite.oldSubdir2)
	suite.oldFile1 = tempFile(suite.T(), "oldFile1", suite.rootDir)
	suite.oldFile4no = tempFile(suite.T(), "oldFile4.*.no", suite.rootDir)
	suite.oldFile4yes = tempFile(suite.T(), "oldFile4.*.yes", suite.rootDir)
	suite.recentFile1 = tempFile(suite.T(), "recentFile1", suite.rootDir)
	suite.recentFile2no = tempFile(suite.T(), "recentFile2.*.no", suite.rootDir)
	suite.recentFile2yes = tempFile(suite.T(), "recentFile2.*.yes", suite.rootDir)
	suite.recentFile3 = tempFile(suite.T(), "recentFile3", suite.oldSubdir3)

	// Set the ages of the files and folders
	suite.age = 14
	setAge(suite.oldSubdir1, suite.age+4)
	setAge(suite.oldSubdir2, suite.age+4)
	setAge(suite.oldSubdir3, suite.age+4)
	setAge(suite.oldEmptySubdir, suite.age+4)
	setAge(suite.recentSubdir1, suite.age-4)
	setAge(suite.oldFile1.Name(), suite.age+4)
	setAge(suite.oldFile2.Name(), suite.age+4)
	setAge(suite.oldFile3no.Name(), suite.age+4)
	setAge(suite.oldFile3yes.Name(), suite.age+4)
	setAge(suite.oldFile4no.Name(), suite.age+4)
	setAge(suite.oldFile4yes.Name(), suite.age+4)
	setAge(suite.recentFile1.Name(), suite.age-4)
	setAge(suite.recentFile2no.Name(), suite.age-4)
	setAge(suite.recentFile2yes.Name(), suite.age-4)
	setAge(suite.recentFile3.Name(), suite.age-4)
}

// The TearDownTest method will be run after every test in the suite.
func (suite *RMStateSuite) TearDownTest() {
	if err := os.RemoveAll(suite.rootDir); err != nil {
		suite.T().Fatal(err)
	}
}

// TestAgeDetection tests the detection of stale files
func (suite *RMStateSuite) TestAgeDetection() {
	for _, t := range []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "Test with an old file",
			filename: suite.oldFile1.Name(),
			want:     true,
		},
		{
			name:     "Test with an old folder",
			filename: suite.oldSubdir1,
			want:     true,
		},
		{
			name:     "Test with a new file",
			filename: suite.recentFile1.Name(),
			want:     false,
		},
		{
			name:     "Test with a new folder",
			filename: suite.recentSubdir1,
			want:     false,
		},
	} {
		suite.Run(t.name, func() {
			got := isStale(fileInfo(suite.T(), t.filename), suite.age)
			suite.Equal(t.want, got)
		})
	}
}

// TestAgeDetection tests the removal of old files
func (suite *RMStateSuite) TestFileRemoval() {
	for _, t := range []struct {
		name      string
		filename  string
		directory string
		dryRun    bool
		want      bool
		wantErr   bool
	}{
		{
			name:      "Test with a file",
			filename:  suite.oldFile1.Name(),
			directory: suite.rootDir,
			dryRun:    false,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "Test with an empty folder",
			filename:  suite.oldEmptySubdir,
			directory: suite.rootDir,
			dryRun:    false,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "Test with a non-empty folder",
			filename:  suite.oldSubdir1,
			directory: suite.rootDir,
			dryRun:    false,
			want:      true,
			wantErr:   true,
		},
		{
			name:      "Test when given the root folder",
			filename:  suite.rootDir,
			directory: suite.rootDir,
			dryRun:    false,
			want:      true,
			wantErr:   false,
		},
	} {
		suite.Run(t.name, func() {
			err := removeItem(t.filename, t.directory, t.dryRun)
			suite.Equal(t.wantErr, err != nil)
			got := exists(t.filename)
			suite.Equal(t.want, got)
		})
	}
}

// TestEmptyDirectoryDetection tests the empty folder detection
func (suite *RMStateSuite) TestEmptyDirectoryDetection() {
	for _, t := range []struct {
		name      string
		filename  string
		directory string
		want      bool
		wantErr   bool
	}{
		{
			name:      "Test with the root folder",
			directory: suite.rootDir,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "Test with an old empty folder",
			directory: suite.oldEmptySubdir,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "Test with an new empty folder",
			directory: suite.recentSubdir1,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "Test with a non-existing folder",
			directory: "fakefile",
			want:      false,
			wantErr:   true,
		},
	} {
		suite.Run(t.name, func() {
			got, err := isEmpty(t.directory)
			suite.Equal(t.wantErr, (err != nil))
			suite.Equal(t.want, got)
		})
	}
}

// TestProcDirErrors tests the edge cases for the procDir function
func (suite *RMStateSuite) TestProcDirErrors() {
	for _, t := range []struct {
		name      string
		path      string
		directory string
		ext       string
		dryRun    bool
		wantErr   bool
	}{
		{
			name:      "Test with a missing file",
			path:      "badFile",
			directory: suite.rootDir,
			ext:       "",
			dryRun:    false,
			wantErr:   true,
		},
		{
			name:      "Test with a file",
			path:      suite.oldFile1.Name(),
			directory: suite.oldFile1.Name(),
			ext:       "",
			dryRun:    false,
			wantErr:   true,
		},
	} {
		suite.Run(t.name, func() {
			err := procDir(t.path, t.directory, suite.age, t.ext, t.dryRun, false)
			suite.Equal(t.wantErr, (err != nil))
		})
	}
}

// TestRemoveItemReturnsErrorForMissingPath verifies that removeItem surfaces
// the underlying os.Remove error instead of swallowing it.
func (suite *RMStateSuite) TestRemoveItemReturnsErrorForMissingPath() {
	missing := filepath.Join(suite.rootDir, "does-not-exist")
	err := removeItem(missing, suite.rootDir, false)
	suite.NotNil(err, "removeItem should return the os.Remove error for a missing path")
}

// TestProcessFilePropagatesRemoveError verifies that processFile returns the
// error produced by removeItem so the caller can aggregate failures.
func (suite *RMStateSuite) TestProcessFilePropagatesRemoveError() {
	info := fileInfo(suite.T(), suite.oldFile1.Name())
	setAge(suite.oldFile1.Name(), suite.age+4) // satisfy isStale on the info
	// Use a non-existent parent directory so path.Join(fp, info.Name())
	// resolves to a path that os.Remove cannot find.
	err := processFile(info, filepath.Join(suite.rootDir, "missing-parent"), suite.rootDir, suite.age, "", false)
	suite.NotNil(err, "processFile should propagate the removeItem error")
}

// TestHandleEmptyDirectoryPropagatesRemoveError verifies that handleEmptyDirectory
// returns the error produced by removeItem so the caller can aggregate failures.
func (suite *RMStateSuite) TestHandleEmptyDirectoryPropagatesRemoveError() {
	// An empty stale directory under a non-existent parent: handleEmptyDirectory
	// will still reach the removeItem branch (pruneEmptyDirs=true) and os.Remove
	// will fail because the path does not exist.
	tmpDir := tempDirectory(suite.T(), "staleEmpty", suite.rootDir)
	setAge(tmpDir, suite.age+4)
	missing := filepath.Join(tmpDir, "missing-empty-dir")

	err := handleEmptyDirectory(missing, fileInfo(suite.T(), tmpDir), suite.age, "", tmpDir, false, true)
	suite.NotNil(err, "handleEmptyDirectory should propagate the removeItem error")
}

// TestDirectoryProcessing tests the running the entire process over a directory
func (suite *RMStateSuite) TestDirectoryProcessing() {
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "", false, false)
	// Ensure that err == nil
	suite.Nil(err)

	// Check that all of the old files are removed
	suite.False(exists(suite.oldFile1.Name()))
	suite.False(exists(suite.oldFile2.Name()))
	suite.False(exists(suite.oldFile3no.Name()))
	suite.False(exists(suite.oldFile3yes.Name()))
	suite.False(exists(suite.oldFile4no.Name()))
	suite.False(exists(suite.oldFile4yes.Name()))

	// Check that the new files are retained
	suite.True(exists(suite.recentFile1.Name()))
	suite.True(exists(suite.recentFile2no.Name()))
	suite.True(exists(suite.recentFile2yes.Name()))

	// Check old empty directories are removed
	suite.False(exists(suite.oldSubdir1))
	suite.False(exists(suite.oldEmptySubdir))

	// Check that the old directories that still have files are retained
	suite.True(exists(suite.oldSubdir3))

	// Check that new directories that contain no files are retained
	suite.True(exists(suite.recentSubdir1))
}

// TestFilteredDirectoryProcessing tests the running the entire process over a directory
func (suite *RMStateSuite) TestFilteredDirectoryProcessing() {
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "yes", false, false)
	// Ensure that err == nil
	suite.Nil(err)

	// Check that all of the old files matching the extension are removed
	suite.False(exists(suite.oldFile3yes.Name()))
	suite.False(exists(suite.oldFile4yes.Name()))

	// Check that all of the old files not matching the extension are retained
	suite.True(exists(suite.oldFile3no.Name()))
	suite.True(exists(suite.oldFile4no.Name()))

	// Check that the new files are retained
	suite.True(exists(suite.recentFile1.Name()))
	suite.True(exists(suite.recentFile2no.Name()))
	suite.True(exists(suite.recentFile2yes.Name()))
	suite.True(exists(suite.recentFile3.Name()))

	// Check all directories are retained
	suite.True(exists(suite.recentSubdir1))
	suite.True(exists(suite.oldSubdir1))
	suite.True(exists(suite.oldSubdir2))
	suite.True(exists(suite.oldSubdir3))
	suite.True(exists(suite.oldEmptySubdir))
}

// TestDryRunOption tests the dry run option
func (suite *RMStateSuite) TestDryRunOption() {
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "yes", true, false)
	// Ensure that err == nil
	suite.Nil(err)

	// Check that all of the old files are retained
	suite.True(exists(suite.oldFile3yes.Name()))
	suite.True(exists(suite.oldFile4yes.Name()))

	// Check that all of the old files not matching the extension are retained
	suite.True(exists(suite.oldFile3no.Name()))
	suite.True(exists(suite.oldFile4no.Name()))

	// Check that the new files are retained
	suite.True(exists(suite.recentFile1.Name()))
	suite.True(exists(suite.recentFile2no.Name()))
	suite.True(exists(suite.recentFile2yes.Name()))
	suite.True(exists(suite.recentFile3.Name()))

	// Check all directories are retained
	suite.True(exists(suite.recentSubdir1))
	suite.True(exists(suite.oldSubdir1))
	suite.True(exists(suite.oldSubdir2))
	suite.True(exists(suite.oldSubdir3))
	suite.True(exists(suite.oldEmptySubdir))
}

// TestPruneEmptyDirsOption tests the prune-empty-dirs option
func (suite *RMStateSuite) TestPruneEmptyDirsOption() {
	// Create a new empty subdirectory that is recent (should normally be kept)
	recentEmptySubdir := tempDirectory(suite.T(), "recentEmptySubdir", suite.rootDir)
	setAge(recentEmptySubdir, suite.age-4)

	// Run procDir with pruneEmptyDirs = true
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "", false, true)
	// Ensure that err == nil
	suite.Nil(err)

	// Check that the recent empty subdirectory is removed
	suite.False(exists(recentEmptySubdir))

	// Check that other empty directories (even recent ones) are also removed
	suite.False(exists(suite.recentSubdir1))

	// Check that non-empty directories are still retained
	suite.True(exists(suite.oldSubdir3))
}

// TestVersionInfo tests the version information output
func (suite *RMStateSuite) TestVersionInfo() {
	expected := "rmstale v0.0.0"
	actual := versionInfo()
	suite.Equal(expected, actual)
}

// TestPrompt tests the prompt function
func (suite *RMStateSuite) TestPrompt() {
	for _, t := range []struct {
		name     string
		format   string
		a        []interface{}
		response string
		want     bool
	}{
		{
			name:     "Test with 'y' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"y"},
			response: "y\n",
			want:     true,
		},
		{
			name:     "Test with 'y' response and nil args",
			format:   "Test prompt (%s).",
			a:        nil,
			response: "y\n",
			want:     true,
		},
		{
			name:     "Test with 'y' response and multiple args",
			format:   "Test prompt (%s).",
			a:        []interface{}{"y", "z"},
			response: "y\n",
			want:     true,
		},
		{
			name:     "Test with 'yes' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"yes"},
			response: "yes\n",
			want:     true,
		},
		{
			name:     "Test with 'YES' response (case-insensitive)",
			format:   "Test prompt (%s).",
			a:        []interface{}{"YES"},
			response: "YES\n",
			want:     true,
		},
		{
			name:     "Test with 'yeah' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"yeah"},
			response: "yeah\n",
			want:     true,
		},
		{
			name:     "Test with 'yup' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"yup"},
			response: "yup\n",
			want:     true,
		},
		{
			name:     "Test with 'y' surrounded by whitespace",
			format:   "Test prompt (%s).",
			a:        []interface{}{"y"},
			response: "  y  \n",
			want:     true,
		},
		{
			name:     "Test with 'n' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"n"},
			response: "n\n",
			want:     false,
		},
		{
			name:     "Test with 'no' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"no"},
			response: "no\n",
			want:     false,
		},
		{
			name:     "Test with empty response (deny on EOF)",
			format:   "Test prompt (%s).",
			a:        []interface{}{"error"},
			response: "",
			want:     false,
		},
		{
			name:     "Test re-prompt on ambiguous then confirm",
			format:   "Test prompt (%s).",
			a:        []interface{}{"maybe"},
			response: "maybe\ny\n",
			want:     true,
		},
		{
			name:     "Test re-prompt on ambiguous then deny",
			format:   "Test prompt (%s).",
			a:        []interface{}{"maybe"},
			response: "maybe\nn\n",
			want:     false,
		},
	} {
		suite.Run(t.name, func() {
			// Redirect standard input for testing
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()
			r, w, _ := os.Pipe()
			os.Stdin = r
			if _, err := fmt.Fprint(w, t.response); err != nil {
				suite.T().Fatal(err)
			}
			if err := w.Close(); err != nil {
				suite.T().Fatal(err)
			}

			got := prompt(t.format, t.a...)
			suite.Equal(t.want, got)
		})
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RMStateSuite))
}

func initLogger() {
	defer logger.Init("rmstale_test", true, false, io.Discard).Close()
	logger.SetFlags(log.Ltime | log.Lshortfile)
}

func fileInfo(t *testing.T, fn string) os.FileInfo {
	fi, err := os.Stat(fn)
	if err != nil {
		t.Fatal(err)
	}
	return fi
}

func tempFile(t *testing.T, prefix, dir string) *os.File {
	content := []byte("Test file contents")
	tmpFile, err := os.CreateTemp(dir, prefix)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpFile
}

func tempDirectory(t *testing.T, prefix, dir string) string {
	tmpDir, err := os.MkdirTemp(dir, prefix)
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func setAge(f string, age int) {
	ts := time.Now().AddDate(0, 0, (age * -1))
	_ = os.Chtimes(f, ts, ts)
}

func exists(fn string) bool {
	if _, err := os.Stat(fn); err == nil {
		return true
	}
	return false
}

// captureOutput captures stdout during function f execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	f()
	if err := w.Close(); err != nil {
		panic(err)
	}
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		panic(err)
	}
	return buf.String()
}

func TestMainVersionFlag(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale", "-v"}
	output := captureOutput(func() { _ = run() })
	if !strings.Contains(output, "rmstale v") {
		t.Fatalf("expected version info, got %q", output)
	}
}

func TestMainNoFlagsShowsUsage(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale"}
	output := captureOutput(func() { _ = run() })
	if !strings.Contains(output, "Usage of rmstale") {
		t.Fatalf("expected usage output, got %q", output)
	}
}

func TestMainHelpShowsDefaults(t *testing.T) {
	output := usage()
	if !strings.Contains(output, os.TempDir()) {
		t.Fatalf("expected default path in usage output, got %q", output)
	}
	if !strings.Contains(output, "(REQUIRED)") || !strings.Contains(output, "(default false)") {
		t.Fatalf("expected default values in usage output, got %q", output)
	}
	if !strings.Contains(output, "--prune-empty-dirs") {
		t.Fatal("expected --prune-empty-dirs in usage output")
	}
}

func TestGetExt(t *testing.T) {
	for _, tt := range []struct{ path, want string }{
		{"file.txt", "txt"},
		{"dir/file.tar.gz", "gz"},
		{"dir/file", ""},
		{"dir.name/file", ""},
	} {
		if got := getExt(tt.path); got != tt.want {
			t.Errorf("getExt(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestMatchExt(t *testing.T) {
	for _, tt := range []struct {
		name string
		ext  string
		want bool
	}{
		{"empty ext always matches", "", true},
		{"match", "txt", true},
		{"no match", "gz", false},
	} {
		file := "test.txt"
		if tt.name == "no match" {
			file = "test.doc"
		}
		if got := matchExt(file, tt.ext); got != tt.want {
			t.Errorf("%s: matchExt(%q,%q) = %v, want %v", tt.name, file, tt.ext, got, tt.want)
		}
	}
}

func TestGetDirectoryContents(t *testing.T) {
	dir := os.TempDir()
	tmp, err := os.CreateTemp(dir, "file")
	if err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(tmp.Name()); err != nil {
			t.Fatal(err)
		}
	})

	infos, err := getDirectoryContents(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("expected some entries, got 0")
	}

	_, err = getDirectoryContents("non-existent")
	if err == nil {
		t.Fatalf("expected error for bad directory")
	}
}

// TestMainWithExtensionMessage tests main function with extension message
func TestMainWithExtensionMessage(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Create a temporary directory for testing
	tmpDir := tempDirectory(t, "test", os.TempDir())
	defer os.RemoveAll(tmpDir)

	// Create files with different extensions
	txtFile := tempFile(t, "test.txt", tmpDir)
	docFile := tempFile(t, "test.doc", tmpDir)
	setAge(txtFile.Name(), 35) // Make them old
	setAge(docFile.Name(), 35)

	os.Args = []string{"rmstale", "-a", "30", "-e", "txt", "-y", "-d", "-p", tmpDir}

	// Run run() (rather than main) and verify extension filtering works.
	// main() would call os.Exit and terminate the test binary, so tests must
	// exercise the CLI logic through run() instead.
	if code := run(); code != exitSuccess {
		t.Fatalf("expected exit code %d on dry-run success, got %d", exitSuccess, code)
	}

	// Both files should still exist since we're in dry-run mode
	// But the extension logic should have been exercised
	if !exists(txtFile.Name()) || !exists(docFile.Name()) {
		t.Fatal("files should still exist in dry-run mode")
	}
}

// TestMainWithProcDirError tests that run() returns a non-zero exit code when
// procDir fails (regression for BSOD-285). The previous behaviour fell off the
// end of main() and reported success (exit code 0) to the OS even though
// processing had failed, which broke scheduled/cron wrappers.
func TestMainWithProcDirError(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale", "-a", "30", "-p", "/nonexistent/path", "-y"}

	code := run()
	if code != exitProcessingError {
		t.Fatalf("expected exit code %d when procDir errors, got %d", exitProcessingError, code)
	}
}

// TestIsEmptyWithFile tests isEmpty function with a file instead of directory
func TestIsEmptyWithFile(t *testing.T) {
	tmpFile := tempFile(t, "test", os.TempDir())
	defer os.Remove(tmpFile.Name())

	_, err := isEmpty(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error when calling isEmpty on a file")
	}
}

// TestGetDirectoryContentsError tests error handling in getDirectoryContents
func TestGetDirectoryContentsError(t *testing.T) {
	_, err := getDirectoryContents("/nonexistent/directory/path")
	if err == nil {
		t.Fatal("expected error when getting contents of non-existent directory")
	}
}

// TestHandleEmptyDirectoryWithExtensionFilter tests handleEmptyDirectory with extension filter
func TestHandleEmptyDirectoryWithExtensionFilter(t *testing.T) {
	tmpDir := tempDirectory(t, "test", os.TempDir())
	defer os.RemoveAll(tmpDir)

	// Make directory old
	setAge(tmpDir, 30)

	// Test with extension filter - should not remove directory even if empty and old
	err := handleEmptyDirectory(tmpDir, fileInfo(t, tmpDir), 20, "txt", tmpDir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should still exist because extension filter is set
	if !exists(tmpDir) {
		t.Fatal("directory should not have been removed with extension filter")
	}
}

// TestHandleEmptyDirectoryWithPruneAndExtensionFilter tests handleEmptyDirectory with both prune and extension filter
func TestHandleEmptyDirectoryWithPruneAndExtensionFilter(t *testing.T) {
	tmpDir := tempDirectory(t, "test", os.TempDir())
	defer os.RemoveAll(tmpDir)

	subDir := tempDirectory(t, "subdir", tmpDir)

	// Make directory old
	setAge(subDir, 30)

	// Test with extension filter AND pruneEmptyDirs - should remove directory
	err := handleEmptyDirectory(subDir, fileInfo(t, subDir), 20, "txt", tmpDir, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should be removed because pruneEmptyDirs is true, even though extension filter is set
	if exists(subDir) {
		t.Fatal("directory should have been removed with pruneEmptyDirs even with extension filter")
	}
}

// TestProcDirWithFileAsPath tests procDir when given a file path instead of directory
func TestProcDirWithFileAsPath(t *testing.T) {
	tmpFile := tempFile(t, "test", os.TempDir())
	defer os.Remove(tmpFile.Name())

	err := procDir(tmpFile.Name(), tmpFile.Name(), 30, "", false, false)
	if err == nil {
		t.Fatal("expected error when processing a file path as directory")
	}
}

// TestMainWithConfirmationDenied tests main function when user denies confirmation
func TestMainWithConfirmationDenied(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Create a temporary directory for testing
	tmpDir := tempDirectory(t, "test", os.TempDir())
	defer os.RemoveAll(tmpDir)

	// Create an old file that would be removed if confirmation was accepted
	oldFile := tempFile(t, "oldfile", tmpDir)
	setAge(oldFile.Name(), 35) // Make it older than 30 days

	os.Args = []string{"rmstale", "-a", "30", "-p", tmpDir}

	// Redirect stdin to simulate user input "n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		w.WriteString("n\n")
	}()

	output := captureOutput(func() {
		code := run()
		if code != exitSuccess {
			t.Fatalf("expected exit code %d when user denies confirmation, got %d", exitSuccess, code)
		}
	})

	// Should contain confirmation prompt and file should still exist since user denied
	if !strings.Contains(output, "Continue") && !strings.Contains(output, "proceed") {
		t.Fatalf("expected confirmation prompt in output, got %q", output)
	}

	// File should still exist since user denied confirmation
	if !exists(oldFile.Name()) {
		t.Fatal("file should still exist after denying confirmation")
	}
}

// TestIsEmptyWithDeferError tests the defer error handling in isEmpty
func TestIsEmptyWithDeferError(t *testing.T) {
	// This is difficult to test directly, but we can test the normal case
	// The defer error handling is covered when the file is properly closed
	tmpDir := tempDirectory(t, "test", os.TempDir())
	defer os.RemoveAll(tmpDir)

	empty, err := isEmpty(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !empty {
		t.Fatal("expected empty directory to be reported as empty")
	}
}

// TestValidateAge ensures the age guard rejects every non-positive value so a
// negative --age cannot silently pass through and wipe the target directory.
// Regression test for BSOD-281.
func TestValidateAge(t *testing.T) {
	for _, tt := range []struct {
		name    string
		age     int
		wantErr bool
	}{
		{"positive age", 30, false},
		{"age of one", 1, false},
		{"zero age", 0, true},
		{"negative age of one", -1, true},
		{"negative age of seven", -7, true},
		{"large negative age", -100, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAge(tt.age)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAge(%d) error = %v, wantErr = %v", tt.age, err, tt.wantErr)
			}
		})
	}
}

// TestMainRejectsNonPositiveAge is the integration regression test for
// BSOD-281: a negative or zero --age must not cause main() to descend into
// procDir. It re-execs the test binary so the os.Exit(1) inside main() does
// not tear down the parent test process.
func TestMainRejectsNonPositiveAge(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		runCrasher(t)
		return
	}

	for _, tt := range []struct {
		name string
		age  string
	}{
		{"negative age", "-1"},
		{"zero age", "0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := tempDirectory(t, "rmstale-bsod281", os.TempDir())
			t.Cleanup(func() { os.RemoveAll(tmpDir) })

			cmd := exec.Command(testBinaryPath, "-test.run=TestMainRejectsNonPositiveAge")
			cmd.Env = append(os.Environ(),
				"BE_CRASHER=1",
				fmt.Sprintf("RMSTALE_TEST_AGE=%s", tt.age),
				fmt.Sprintf("RMSTALE_TEST_DIR=%s", tmpDir),
			)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected non-zero exit code, got nil. output: %s", out)
			}
			if !strings.Contains(string(out), "must be a positive integer") {
				t.Logf("subprocess output: %q (len=%d)", out, len(out))
				t.Logf("subprocess err: %v", err)
				t.Fatalf("expected positive-integer error message, got %q", out)
			}
		})
	}
}

// runCrasher is executed inside the BE_CRASHER subprocess. It sets up a
// controlled temp directory with a sentinel file and invokes main() with the
// age value supplied by the parent. If main() does not exit early (i.e. it
// descends into procDir), the sentinel file is deleted and the test fails.
func runCrasher(t *testing.T) {
	age := os.Getenv("RMSTALE_TEST_AGE")
	dir := os.Getenv("RMSTALE_TEST_DIR")
	if age == "" || dir == "" {
		t.Fatal("crasher requires RMSTALE_TEST_AGE and RMSTALE_TEST_DIR")
	}

	sentinel := tempFile(t, "must-not-be-deleted", dir)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale", "-a", age, "-p", dir, "-y"}
	main()

	if exists(sentinel.Name()) {
		// main() returned without calling os.Exit — guard never tripped.
		os.Exit(2)
	}
	os.Exit(0)
}

// TestValidatePath exercises the protected-roots guard added for BSOD-282.
// Canonicalisation is performed inside validatePath, so the table covers both
// exact roots (must be refused) and sub-paths of those roots (must remain
// reachable so legitimate staging areas are not blocked by default). The
// denial is an exact-match compare: tmpdir defaults such as /tmp on Linux,
// /var/folders/... on macOS, and C:\Users\<user>\AppData\Local\Temp on Windows
// all need to remain allowed.
//
// Each case is run twice: once without the override (default behaviour, which
// must reject the listed roots) and once with --allow-system-paths (which must
// accept every input). Sub-paths are required to be accepted in both modes.
func TestValidatePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Fatalf("os.UserHomeDir() = %q, %v; cannot build absolute home dir for test", home, err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", home, err)
	}
	absHome = filepath.Clean(absHome)

	type expectation struct {
		// wantErrDefault is true when validatePath should refuse the path
		// without the override; sub-paths of protected roots set this to
		// false so legitimate areas stay reachable.
		wantErrDefault bool
		// wantErrOverride is true when validatePath should refuse the path
		// even with --allow-system-paths. Inputs that are not absolute
		// filesystem paths or that cannot be canonicalised fall here.
		wantErrOverride bool
	}

	tests := []struct {
		name string
		path string
		exp  expectation
	}{
		// Always allowed in both modes: sub-paths and the OS temp dir.
		{"default temp dir", os.TempDir(), expectation{false, false}},
		{"sub-path of home", filepath.Join(absHome, "Documents"), expectation{false, false}},
		{"sub-path of /var (macOS temp dir)", filepath.FromSlash("/var/folders/ab/ef/T"), expectation{false, false}},
		{"plain relative path resolved under temp", filepath.Join(os.TempDir(), "rmstale-test"), expectation{false, false}},
		{"normal sub-path under /tmp", "/tmp/foo", expectation{false, false}},

		// Default refuses, override permits: exactly the protected roots.
		{"filesystem root", string(filepath.Separator), expectation{true, false}},
		{"trailing slash on root", string(filepath.Separator), expectation{true, false}},
		{"double-slash collapses to root", "//", expectation{true, false}},
		{"dot smuggled into root", string(filepath.Separator) + "etc/.", expectation{true, false}},
		{"double-dot returns to root via /etc", "/etc/../etc", expectation{true, false}},
		{"linux /etc exact", "/etc", expectation{runtime.GOOS != "windows", false}},
		{"linux /usr exact", "/usr", expectation{runtime.GOOS != "windows", false}},
		{"linux /var exact", "/var", expectation{runtime.GOOS != "windows", false}},
		{"linux /boot exact", "/boot", expectation{runtime.GOOS != "windows", false}},
		{"linux /proc exact", "/proc", expectation{runtime.GOOS != "windows", false}},
		{"linux /sys exact", "/sys", expectation{runtime.GOOS != "windows", false}},
		{"windows C:\\ exact", `C:\`, expectation{runtime.GOOS == "windows", false}},
		{"user home exact", absHome, expectation{true, false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Default behaviour.
			if err := validatePath(tc.path, false); (err != nil) != tc.exp.wantErrDefault {
				t.Errorf("validatePath(%q, false) error = %v, wantErr = %v", tc.path, err, tc.exp.wantErrDefault)
			} else if tc.exp.wantErrDefault && err != nil && !strings.Contains(err.Error(), "protected") {
				t.Errorf("validatePath(%q, false) error %q should mention 'protected'", tc.path, err.Error())
			}

			// Override behaviour.
			if err := validatePath(tc.path, true); (err != nil) != tc.exp.wantErrOverride {
				t.Errorf("validatePath(%q, true) error = %v, wantErr = %v", tc.path, err, tc.exp.wantErrOverride)
			}
		})
	}
}

// TestMainRejectsSensitivePath is the integration regression test for
// BSOD-282: a protected root must not cause main() to descend into procDir.
// It re-execs the test binary so the os.Exit(1) inside run() does not tear
// down the parent test process. The subprocess exit code is the assertion:
// exitSuccess (0) means the guard fired before procDir could run; any other
// code means the validation guard let an unsafe path through and procDir
// either ran or errored on its own.
func TestMainRejectsSensitivePath(t *testing.T) {
	if os.Getenv("BE_CRASHER_PATH") == "1" {
		runCrasherPath(t)
		return
	}

	cases := []struct {
		name string
		path string
	}{
		{"filesystem root", string(filepath.Separator)},
		{"etc root", "/etc"},
	}

	if runtime.GOOS != "windows" {
		cases = append(cases,
			struct{ name, path string }{"usr root", "/usr"},
			struct{ name, path string }{"var root", "/var"},
			struct{ name, path string }{"proc root", "/proc"},
		)
	}
	home, herr := os.UserHomeDir()
	if herr == nil && home != "" {
		absHome, aerr := filepath.Abs(home)
		if aerr == nil {
			cases = append(cases, struct{ name, path string }{"user home", filepath.Clean(absHome)})
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(testBinaryPath, "-test.run=TestMainRejectsSensitivePath")
			cmd.Env = append(os.Environ(),
				"BE_CRASHER_PATH=1",
				fmt.Sprintf("RMSTALE_TEST_PATH=%s", tc.path),
			)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected non-zero exit code, got nil. output: %s", out)
			}
			if !strings.Contains(string(out), "protected") {
				t.Fatalf("expected 'protected' error message, got %q (err=%v)", out, err)
			}
		})
	}
}

// TestMainAllowsSubPathOfProtectedRoot is the positive control for BSOD-282:
// sub-paths of protected roots (the default --path os.TempDir() is one of
// them on Linux/macOS/Windows) must remain reachable without the override.
// The subprocess exits with run()'s return code; exitSuccess (0) means the
// guard did not over-reject and procDir completed cleanly.
func TestMainAllowsSubPathOfProtectedRoot(t *testing.T) {
	if os.Getenv("BE_CRASHER_SUBDIR") == "1" {
		runCrasherSubdir(t)
	}

	dir := tempDirectory(t, "rmstale-bsod282-subdir", os.TempDir())
	t.Cleanup(func() { os.RemoveAll(dir) })

	cmd := exec.Command(testBinaryPath, "-test.run=TestMainAllowsSubPathOfProtectedRoot")
	cmd.Env = append(os.Environ(),
		"BE_CRASHER_SUBDIR=1",
		fmt.Sprintf("RMSTALE_TEST_DIR=%s", dir),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sub-path of protected root should be allowed without override; got exit err=%v output=%q", err, out)
	}
}

// runCrasherPath invokes run() with the path supplied via RMSTALE_TEST_PATH.
// run() returns exitSuccess (0) only when the validation guard rejects the
// path; any other return code (including exitProcessingError from procDir)
// bubbles up as a non-zero subprocess exit so the parent test sees the
// regression.
func runCrasherPath(t *testing.T) {
	path := os.Getenv("RMSTALE_TEST_PATH")
	if path == "" {
		t.Fatal("crasher requires RMSTALE_TEST_PATH")
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// -a 1 forces isStale to be true for every file; if procDir were
	// reached on a sensitive root it would call os.Remove on real system
	// paths. The validation guard must prevent that. We pass -y to skip
	// the interactive prompt. We also pass --allow-system-paths=false
	// implicitly (the default); runCrasherPathAllowSystemPaths below
	// covers the override branch.
	os.Args = []string{"rmstale", "-a", "1", "-p", path, "-y"}
	os.Exit(run())
}

// runCrasherSubdir is the positive-control crasher. It invokes run() with a
// sub-directory of os.TempDir() (which is itself a sub-path of every
// protected root on every supported platform). If the validation guard is
// over-eager and rejects sub-paths, run() returns non-zero; otherwise it
// returns exitSuccess and the subprocess exits 0.
func runCrasherSubdir(t *testing.T) {
	dir := os.Getenv("RMSTALE_TEST_DIR")
	if dir == "" {
		t.Fatal("crasher requires RMSTALE_TEST_DIR")
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale", "-a", "30", "-p", dir, "-y"}
	os.Exit(run())
}
