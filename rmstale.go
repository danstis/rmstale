package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	prompt "github.com/segmentio/go-prompt"
)

// AppVersion controls the application version number
var AppVersion = "0.0.0"
var age int
var folder string

func main() {
	flag.StringVar(&folder, "path", os.TempDir(), "Path to check for stale files.")
	flag.IntVar(&age, "age", 0, "Age in days to check for file modification.")
	confirm := flag.Bool("y", false, "Don't prompt for confirmation.")
	version := flag.Bool("version", false, "Display version information.")

	flag.Parse()

	if *version {
		fmt.Printf("rmstale v%v\n", AppVersion)
		os.Exit(0)
	}

	if age == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !*confirm {
		if ok := prompt.Confirm("WARNING: Will remove files and folders recursively below '%v' older than %v days. Continue?", filepath.FromSlash(folder), age); !ok {
			fmt.Println("Operation not confirmed, exiting.")
			os.Exit(1)
		}
	}

	if err := procDir(folder); err != nil {
		fmt.Printf("Something went wrong: %v", err)
	}
}

func procDir(fp string) error {
	// get the fileInfo for the directory
	di, err := os.Stat(fp)
	if err != nil {
		return err
	}

	// get the directory contents
	contents, err := ioutil.ReadDir(fp)
	if err != nil {
		return err
	}

	for _, item := range contents {
		if item.IsDir() {
			if err := procDir(path.Join(fp, item.Name())); err != nil {
				return err
			}
		} else {
			if isStale(item) {
				removeItem(path.Join(fp, item.Name()))
			}
		}
	}

	empty, err := isEmpty(fp)
	if err != nil {
		return err
	}
	if empty && isStale(di) {
		removeItem(fp)
	}

	return nil
}

// isEmpty checks if a directory is empty.
func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// isStale checks if the file/directory is older than age days old.
func isStale(fi os.FileInfo) bool {
	return fi.ModTime().Before(time.Now().AddDate(0, 0, (age * -1)))
}

// removeItem removes an item from the filesystem.
func removeItem(fp string) {
	if fp == folder {
		fmt.Printf("-Not removing folder '%v' as it is the root folder...\n", filepath.FromSlash(fp))
	}
	fmt.Printf("-Removing '%v'...\n", filepath.FromSlash(fp))
	if err := os.Remove(fp); err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}
}
