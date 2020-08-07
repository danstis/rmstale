package main

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/logger"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	initLogger()
}

func TestAgeDetection(t *testing.T) {

	Convey("Given a file", t, func() {
		file := tempFile(t, "file1", os.TempDir())
		defer os.Remove(file.Name())
		age := 14

		Convey("With a modification date older than the defined age", func() {
			setAge(file.Name(), age+5)

			Convey("It is detected as being stale", func() {
				fi := fileInfo(t, file.Name())
				So(isStale(fi, age), ShouldBeTrue)
			})
		})

		Convey("With a modification date newer than the defined age", func() {
			setAge(file.Name(), age-5)

			Convey("It is not detected as stale", func() {
				fi := fileInfo(t, file.Name())
				So(isStale(fi, age), ShouldBeFalse)
			})
		})
	})
}

func TestFileRemoval(t *testing.T) {

	Convey("Given a file to remove", t, func() {
		tmpFile := tempFile(t, "removeMe", os.TempDir())
		defer os.Remove(tmpFile.Name())
		removeItem(tmpFile.Name(), os.TempDir())

		Convey("The file no longer exists", func() {
			So(exists(tmpFile.Name()), ShouldBeFalse)
		})
	})

	Convey("Given a directory to remove", t, func() {
		tmpDir := tempDirectory(t, "toRemove", os.TempDir())
		defer os.RemoveAll(tmpDir)
		removeItem(tmpDir, os.TempDir())

		Convey("The directory no longer exists", func() {
			So(exists(tmpDir), ShouldBeFalse)
		})
	})

	Convey("Given the root folder to remove", t, func() {
		tmpDir := tempDirectory(t, "toStay", os.TempDir())
		defer os.RemoveAll(tmpDir)
		removeItem(tmpDir, tmpDir)

		Convey("The root folder is not removed", func() {
			So(exists(tmpDir), ShouldBeTrue)
		})
	})
}

func TestEmptyDirectoryDetection(t *testing.T) {

	Convey("Given an empty directory", t, func() {
		tmpDir := tempDirectory(t, "emptyDir", os.TempDir())
		defer os.RemoveAll(tmpDir)

		Convey("It is detected as being empty", func() {
			empty, err := isEmpty(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			So(empty, ShouldBeTrue)
		})
	})

	Convey("Given a directory containing a file", t, func() {
		tmpDir := tempDirectory(t, "emptyDir", os.TempDir())
		defer os.RemoveAll(tmpDir)
		tmpFile := tempFile(t, "abc", tmpDir)
		defer os.Remove(tmpFile.Name())

		Convey("It is detected as not being empty", func() {
			empty, err := isEmpty(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			So(empty, ShouldBeFalse)
		})
	})
}

func TestDirectoryProcessing(t *testing.T) {

	Convey("Given a folder with files and subfolders", t, func() {
		// Create the following structure:
		//		rootDir/
		//			oldSubdir1/
		//				oldFile2
		//			oldSubdir2/
		//				oldFile3.yes
		//				oldFile3.no
		//			oldSubdir3/
		//				recentFile3
		//			recentSubdir1/
		//			oldFile1
		//			oldFile4.no
		//			oldFile4.yes
		//			recentFile1
		//			recentFile2.no
		//			recentFile2.yes

		// Create folder structure
		rootDir := tempDirectory(t, "rootDir", os.TempDir())
		defer os.RemoveAll(rootDir)
		oldSubdir1 := tempDirectory(t, "oldSubdir1", rootDir)
		defer os.RemoveAll(oldSubdir1)
		oldSubdir2 := tempDirectory(t, "oldSubdir2", rootDir)
		defer os.RemoveAll(oldSubdir2)
		oldSubdir3 := tempDirectory(t, "oldSubdir3", rootDir)
		defer os.RemoveAll(oldSubdir3)
		recentSubdir1 := tempDirectory(t, "recentSubdir1", rootDir)
		defer os.RemoveAll(recentSubdir1)

		// Create file structure
		oldFile2 := tempFile(t, "oldFile2", oldSubdir1)
		defer os.Remove(oldFile2.Name())
		oldFile3no := tempFile(t, "oldFile3.*.no", oldSubdir2)
		defer os.Remove(oldFile3no.Name())
		oldFile3yes := tempFile(t, "oldFile3.*.yes", oldSubdir2)
		defer os.Remove(oldFile3yes.Name())
		oldFile1 := tempFile(t, "oldFile1", rootDir)
		defer os.Remove(oldFile1.Name())
		oldFile4no := tempFile(t, "oldFile4.*.no", rootDir)
		defer os.Remove(oldFile4no.Name())
		oldFile4yes := tempFile(t, "oldFile4.*.yes", rootDir)
		defer os.Remove(oldFile4yes.Name())
		recentFile1 := tempFile(t, "recentFile1", rootDir)
		defer os.Remove(recentFile1.Name())
		recentFile2no := tempFile(t, "recentFile2.*.no", rootDir)
		defer os.Remove(recentFile2no.Name())
		recentFile2yes := tempFile(t, "recentFile2.*.yes", rootDir)
		defer os.Remove(recentFile2yes.Name())
		recentFile3 := tempFile(t, "recentFile3", oldSubdir3)
		defer os.Remove(recentFile3.Name())

		// Set the ages of the files and folders
		age := 14
		setAge(oldSubdir1, age+4)
		setAge(oldSubdir2, age+4)
		setAge(oldSubdir3, age+4)
		setAge(recentSubdir1, age-4)
		setAge(oldFile1.Name(), age+4)
		setAge(oldFile2.Name(), age+4)
		setAge(oldFile3no.Name(), age+4)
		setAge(oldFile3yes.Name(), age+4)
		setAge(oldFile4no.Name(), age+4)
		setAge(oldFile4yes.Name(), age+4)
		setAge(recentFile1.Name(), age-4)
		setAge(recentFile2no.Name(), age-4)
		setAge(recentFile2yes.Name(), age-4)
		setAge(recentFile3.Name(), age-4)

		Convey("Told to process unfiltered removals", func() {
			if err := procDir(rootDir, rootDir, age, ""); err != nil {
				t.Fatal(err)
			}

			Convey("Old files are removed", func() {
				So(exists(oldFile1.Name()), ShouldBeFalse)
				So(exists(oldFile2.Name()), ShouldBeFalse)
				So(exists(oldFile3no.Name()), ShouldBeFalse)
				So(exists(oldFile3yes.Name()), ShouldBeFalse)
				So(exists(oldFile4no.Name()), ShouldBeFalse)
				So(exists(oldFile4yes.Name()), ShouldBeFalse)
			})

			Convey("New files are retained", func() {
				So(exists(recentFile1.Name()), ShouldBeTrue)
				So(exists(recentFile2no.Name()), ShouldBeTrue)
				So(exists(recentFile2yes.Name()), ShouldBeTrue)
			})

			Convey("Empty directories that are old and contain no files are removed", func() {
				So(exists(oldSubdir1), ShouldBeFalse)
			})

			Convey("Empty directories that are old and contain files are retained", func() {
				So(exists(oldSubdir3), ShouldBeTrue)
			})

			Convey("Empty directories that are new but contain no files are retained", func() {
				So(exists(recentSubdir1), ShouldBeTrue)
			})
		})

		Convey("Told to process filtered removals", func() {
			if err := procDir(rootDir, rootDir, age, "yes"); err != nil {
				t.Fatal(err)
			}

			Convey("Old files with matching extension are removed", func() {
				So(exists(oldFile3yes.Name()), ShouldBeFalse)
				So(exists(oldFile4yes.Name()), ShouldBeFalse)
			})

			Convey("Old files not matching extension are retained", func() {
				So(exists(oldFile3no.Name()), ShouldBeTrue)
				So(exists(oldFile4no.Name()), ShouldBeTrue)
			})

			Convey("New files are retained", func() {
				So(exists(recentFile1.Name()), ShouldBeTrue)
				So(exists(recentFile2no.Name()), ShouldBeTrue)
				So(exists(recentFile2yes.Name()), ShouldBeTrue)
				So(exists(recentFile3.Name()), ShouldBeTrue)
			})

			Convey("Empty directories that are new but contain no files are retained", func() {
				So(exists(recentSubdir1), ShouldBeTrue)
			})
		})
	})
}

func initLogger() {
	defer logger.Init("rmstale_test", true, false, ioutil.Discard).Close()
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
	tmpFile, err := ioutil.TempFile(dir, prefix)
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
	tmpDir, err := ioutil.TempDir(dir, prefix)
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
