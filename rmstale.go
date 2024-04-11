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
	"strings"
	"time"

	"github.com/google/logger"
)

// AppVersion controls the application version number
var AppVersion = "0.0.0"

const usage = `Usage of rmstale:
  -a, --age 		Period in days before an item is considered stale.
  -d, --dry-run		Runs the process in dry-run mode, no files will be removed.
  -e, --extension	Filter files for a defined file extension.
  -p, --path		Path to a folder to process.
  -v, --version		Displays the version of rmstale that is currently running.
  -y, --confirm		Allows for processing without confirmation prompt, useful for scheduling.
`

func main() {
	flag.Usage = func() { fmt.Print(usage) }

	var (
		folder      string
		age         int
		confirm     bool
		ext         string
		showVersion bool
		extMsg      string
		dryRun      bool
	)
	flag.StringVar(&folder, "p", os.TempDir(), "Path to check for stale files.")
	flag.StringVar(&folder, "path", os.TempDir(), "Path to check for stale files.")
	flag.IntVar(&age, "a", 0, "Age in days to check for file modification.")
	flag.IntVar(&age, "age", 0, "Age in days to check for file modification.")
	flag.BoolVar(&confirm, "y", false, "Don't prompt for confirmation.")
	flag.BoolVar(&confirm, "confirm", false, "Don't prompt for confirmation.")
	flag.StringVar(&ext, "e", "", "Filter files by extension.")
	flag.StringVar(&ext, "extension", "", "Filter files by extension.")
	flag.BoolVar(&showVersion, "v", false, "Display version information.")
	flag.BoolVar(&showVersion, "version", false, "Display version information.")
	flag.BoolVar(&dryRun, "d", false, "Dry run mode, no files will be removed.")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run mode, no files will be removed.")

	// Parse flags
	flag.Parse()

	defer logger.Init("rmstale", true, true, io.Discard).Close()
	logger.SetFlags(log.Ltime)

	// Check if no command-line arguments were provided or if an argument is provided without a '-'
	if flag.NFlag() == 0 && len(flag.Args()) == 0 || len(flag.Args()) > 0 && flag.Arg(0)[0] != '-' {
		flag.Usage()
		return
	}

	if ext != "" {
		extMsg = fmt.Sprintf(" with extension '%v'", ext)
	}

	if showVersion {
		fmt.Println(versionInfo())
		return
	}

	if age == 0 {
		flag.Usage()
		return
	}

	if !confirm && !dryRun && !prompt("WARNING: Will remove files and folders recursively below '%v'%s older than %v days.", filepath.FromSlash(folder), extMsg, age) {
		logger.Warning("Operation not confirmed, exiting.")
		return
	}

	logger.Infof("rmstale started against folder '%v'%s for contents older than %v days.", filepath.FromSlash(folder), extMsg, age)

	if err := procDir(folder, folder, age, ext, dryRun); err != nil {
		logger.Errorf("Something went wrong: %v", err)
	}
}

// versionInfo returns the version information of the rmstale application
func versionInfo() string {
	return fmt.Sprintf("rmstale v%v", AppVersion)
}

// prompt prompts the user for confirmation before proceeding.
// It returns true if the user confirms, false otherwise.
func prompt(format string, a ...interface{}) bool {
	fmt.Printf(format+" Continue? (y/n) ", a...)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		logger.Errorf("Failed to read user input: %v", err)
		return false
	}
	return strings.ToLower(response) == "y"
}

// procDir recursively processes a directory and removes stale files.
// It takes the file path (fp) of the directory to process, the root folder (rootFolder) for reference,
// the age (age) in days to determine staleness, and the file extension (ext) to filter files.
// It returns an error if any operation fails.
func procDir(fp, rootFolder string, age int, ext string, dryRun bool) error {
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
			if err := procDir(path.Join(fp, item.Name()), rootFolder, age, ext, dryRun); err != nil {
				return err
			}
		} else {
			if isStale(item, age) && matchExt(item.Name(), ext) {
				removeItem(path.Join(fp, item.Name()), rootFolder, dryRun)
			}
		}
	}

	empty, err := isEmpty(fp)
	if err != nil {
		return err
	}
	if empty && isStale(di, age) && ext == "" {
		removeItem(fp, rootFolder, dryRun)
	}

	return nil
}

// isEmpty checks if a directory is empty.
// It returns true if the directory is empty, false otherwise.
// An error is returned if there was a problem opening or reading the directory.
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
func isStale(fi os.FileInfo, age int) bool {
	return fi.ModTime().Before(time.Now().AddDate(0, 0, (age * -1)))
}

// removeItem removes an item from the filesystem.
func removeItem(fp, rootFolder string, dryRun bool) {
	if fp == rootFolder {
		logger.Infof("Not removing folder '%v' as it is the root folder...\n", filepath.FromSlash(fp))
		return
	}
	if dryRun {
		logger.Infof("[DRY RUN] '%v' would be removed...", filepath.FromSlash(fp))
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
