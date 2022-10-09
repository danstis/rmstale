package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/google/logger"
	prompt "github.com/segmentio/go-prompt"
)

// AppVersion controls the application version number
var AppVersion = "0.0.0"

const usage = `Usage of rmstale:
  -a, --age 		Period in days before an item is considered stale.
  -p, --path		Path to a folder to process.
  -y, --confirm		Allows for processing without confirmation prompt, useful for scheduling.
  -v, --version		Displays the version of rmstale that is currently running.
  -e, --extension	Filter files for a defined file extension.
`

func main() {
	var (
		folder  string
		age     int
		confirm bool
		ext     string
		version bool
		extMsg  string
	)
	flag.StringVar(&folder, "p", os.TempDir(), "Path to check for stale files.")
	flag.StringVar(&folder, "path", os.TempDir(), "Path to check for stale files.")
	flag.IntVar(&age, "a", 0, "Age in days to check for file modification.")
	flag.IntVar(&age, "age", 0, "Age in days to check for file modification.")
	flag.BoolVar(&confirm, "y", false, "Don't prompt for confirmation.")
	flag.BoolVar(&confirm, "confirm", false, "Don't prompt for confirmation.")
	flag.StringVar(&ext, "e", "", "Filter files by extension.")
	flag.StringVar(&ext, "extension", "", "Filter files by extension.")
	flag.BoolVar(&version, "v", false, "Display version information.")
	flag.BoolVar(&version, "version", false, "Display version information.")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	defer logger.Init("rmstale", true, true, io.Discard).Close()
	logger.SetFlags(log.Ltime)

	flag.Parse()

	if ext != "" {
		extMsg = fmt.Sprintf(" with extension '%v'", ext)
	}

	if version {
		fmt.Printf("rmstale v%v\n", AppVersion)
		return
	}

	if age == 0 {
		flag.PrintDefaults()
		return
	}

	if !confirm {
		if ok := prompt.Confirm("WARNING: Will remove files and folders recursively below '%v'%s older than %v days. Continue?", filepath.FromSlash(folder), extMsg, age); !ok {
			logger.Warning("Operation not confirmed, exiting.")
			return
		}
	}

	logger.Infof("rmstale started against folder '%v'%s for contents older than %v days.", filepath.FromSlash(folder), extMsg, age)

	if err := procDir(folder, folder, age, ext); err != nil {
		logger.Errorf("Something went wrong: %v", err)
	}
}

func procDir(fp, rootFolder string, age int, ext string) error {
	// get the fileInfo for the directory
	di, err := os.Stat(fp)
	if err != nil {
		return err
	}

	// get the directory contents
	contents, err := os.ReadDir(fp)
	if err != nil {
		return err
	}
	infos := make([]fs.FileInfo, 0, len(contents))
	for _, entry := range contents {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		infos = append(infos, info)
	}

	for _, item := range infos {
		if item.IsDir() {
			if err := procDir(path.Join(fp, item.Name()), rootFolder, age, ext); err != nil {
				return err
			}
		} else {
			if isStale(item, age) && matchExt(item.Name(), ext) {
				removeItem(path.Join(fp, item.Name()), rootFolder)
			}
		}
	}

	empty, err := isEmpty(fp)
	if err != nil {
		return err
	}
	if empty && isStale(di, age) && ext == "" {
		removeItem(fp, rootFolder)
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
	f.Sync()
	return false, err
}

// isStale checks if the file/directory is older than age days old.
func isStale(fi os.FileInfo, age int) bool {
	return fi.ModTime().Before(time.Now().AddDate(0, 0, (age * -1)))
}

// removeItem removes an item from the filesystem.
func removeItem(fp, rootFolder string) {
	if fp == rootFolder {
		logger.Infof("Not removing folder '%v' as it is the root folder...\n", filepath.FromSlash(fp))
		return
	}
	logger.Infof("Removing '%v'...", filepath.FromSlash(fp))
	if err := os.Remove(fp); err != nil {
		logger.Errorf("%v", err)
	}
}

// getExt returns the file extension of the presented path.
func getExt(path string) string {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return ""
}

// matchExt returns true if the file name specified matches the extension specified.
func matchExt(name, ext string) bool {
	if ext == "" {
		return true
	}
	return getExt(name) == ext
}
