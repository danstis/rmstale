package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/logger"
	"github.com/stretchr/testify/suite"
)

func init() {
	initLogger()
}

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
	}{
		{
			name:      "Test with a file",
			filename:  suite.oldFile1.Name(),
			directory: suite.rootDir,
			dryRun:    false,
			want:      false,
		},
		{
			name:      "Test with an empty folder",
			filename:  suite.oldEmptySubdir,
			directory: suite.rootDir,
			dryRun:    false,
			want:      false,
		},
		{
			name:      "Test with a non-empty folder",
			filename:  suite.oldSubdir1,
			directory: suite.rootDir,
			dryRun:    false,
			want:      true,
		},
		{
			name:      "Test when given the root folder",
			filename:  suite.rootDir,
			directory: suite.rootDir,
			dryRun:    false,
			want:      true,
		},
	} {
		suite.Run(t.name, func() {
			removeItem(t.filename, t.directory, t.dryRun)
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
			err := procDir(t.path, t.directory, suite.age, t.ext, t.dryRun)
			suite.Equal(t.wantErr, (err != nil))
		})
	}
}

// TestDirectoryProcessing tests the running the entire process over a directory
func (suite *RMStateSuite) TestDirectoryProcessing() {
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "", false)
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
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "yes", false)
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
	err := procDir(suite.rootDir, suite.rootDir, suite.age, "yes", true)
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
			name:     "Test with 'n' response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"n"},
			response: "n\n",
			want:     false,
		},
		{
			name:     "Test with invalid response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"invalid"},
			response: "invalid\n",
			want:     false,
		},
		{
			name:     "Test with error response",
			format:   "Test prompt (%s).",
			a:        []interface{}{"error"},
			response: "",
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
