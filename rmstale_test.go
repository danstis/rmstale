package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAgeDetection(t *testing.T) {

	Convey("Given a file", t, func() {
		file := tempFile(t, "file1")
		defer os.Remove(file.Name())
		age := 14

		Convey("With a modification date older than the defined age", func() {
			setAge(t, file, age+5)

			Convey("It is detected as being stale", func() {
				fi := fileInfo(t, file.Name())
				So(isStale(fi, age), ShouldBeTrue)
			})

		})

		Convey("With a modification date newer than the defined age", func() {
			setAge(t, file, age-5)

			Convey("It is not detected as stale", func() {
				fi := fileInfo(t, file.Name())
				So(isStale(fi, age), ShouldBeFalse)
			})

		})

	})

}

func TestFileRemoval(t *testing.T) {

	Convey("Given a file to remove", t, func() {
		tmpFile := tempFile(t, "removeMe")
		defer os.Remove(tmpFile.Name())
		removeItem(tmpFile.Name(), os.TempDir())

		Convey("The file no longer exists", func() {
			So(exists(tmpFile.Name()), ShouldBeFalse)
		})

	})

	Convey("Given a directory to remove", t, func() {
		tmpDir := tempDirectory(t, "toRemove")
		defer os.RemoveAll(tmpDir) // clean up
		removeItem(tmpDir, os.TempDir())

		Convey("The directory no longer exists", func() {
			So(exists(tmpDir), ShouldBeFalse)
		})

	})

	Convey("Given the root folder to remove", t, func() {
		tmpDir := tempDirectory(t, "toStay")
		defer os.RemoveAll(tmpDir) // clean up
		removeItem(tmpDir, tmpDir)

		Convey("The root folder is not removed", func() {
			So(exists(tmpDir), ShouldBeTrue)
		})

	})

}

func fileInfo(t *testing.T, fn string) os.FileInfo {
	fi, err := os.Stat(fn)
	if err != nil {
		t.Fatal(err)
	}
	return fi
}

func tempFile(t *testing.T, p string) *os.File {
	content := []byte("Test file contents")
	tmpFile, err := ioutil.TempFile(os.TempDir(), p)
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

func tempDirectory(t *testing.T, p string) string {
	tmpDir, err := ioutil.TempDir(os.TempDir(), p)
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func setAge(t *testing.T, f *os.File, age int) {
	ts := time.Now().AddDate(0, 0, (age * -1))
	os.Chtimes(f.Name(), ts, ts)
}

func exists(fn string) bool {
	if _, err := os.Stat(fn); err == nil {
		return true
	}
	return false
}
