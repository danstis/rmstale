// Package main implements rmstale, a tool for removing stale files and directories.
package main

import (
	"bufio"
	"errors"
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

func usage() string {
	return fmt.Sprintf(`Usage of rmstale:
  -a, --age             Period in days before an item is considered stale. (REQUIRED)
  -d, --dry-run         Runs the process in dry-run mode. No files will be removed, but the tool will log the files that would be deleted. (default %v)
  -e, --extension       Filter files for a defined file extension. This flag only applies to files, not directories. (default %q)
  -p, --path            Path to a folder to process. (default %s)
  -v, --version         Displays the version of rmstale that is currently running. (default %v)
  -y, --confirm         Allows for processing without confirmation prompt, useful for scheduling. (default %v)
  --prune-empty-dirs    Remove empty directories even if they are not stale. (default %v)
`, false, "", filepath.FromSlash(os.TempDir()), false, false, false)
}

func main() {
	flag.Usage = func() { fmt.Print(usage()) }

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
	var pruneEmptyDirs bool
	flag.BoolVar(&pruneEmptyDirs, "prune-empty-dirs", false, "Remove empty directories even if they are not stale.")

	// Parse flags
	flag.Parse()

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

	if err := validateAge(age); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		flag.Usage()
		os.Exit(1)
	}

	defer logger.Init("rmstale", true, true, io.Discard).Close()
	logger.SetFlags(log.Ltime)

	if !confirm && !dryRun && !prompt("WARNING: Will remove files and folders recursively below '%v'%s older than %v days.", filepath.FromSlash(folder), extMsg, age) {
		logger.Warning("Operation not confirmed, exiting.")
		return
	}

	logger.Infof("rmstale started against folder '%v'%s for contents older than %v days.", filepath.FromSlash(folder), extMsg, age)

	if err := procDir(folder, folder, age, ext, dryRun, pruneEmptyDirs); err != nil {
		logger.Errorf("Something went wrong: %v", err)
	}
}

// versionInfo returns the version information of the rmstale application
func versionInfo() string {
	return fmt.Sprintf("rmstale v%v", AppVersion)
}

// validateAge ensures the supplied --age is a positive integer. Any value <= 0
// is rejected because negative ages invert the staleness comparison and cause
// the tool to treat every file as stale (i.e. delete the entire target tree).
func validateAge(age int) error {
	if age <= 0 {
		return fmt.Errorf("--age must be a positive integer (got %d)", age)
	}
	return nil
}

// prompt prompts the user for confirmation before proceeding.
// It returns true if the user confirms, false otherwise.
// Affirmative tokens: "y", "yes", "yeah", "yup" (case-insensitive, whitespace-trimmed).
// Negative tokens:   "n", "no", "nope", "nah"  (case-insensitive, whitespace-trimmed).
// Any other input is treated as ambiguous and re-prompts until the user gives a
// clear answer or input ends (EOF), in which case it returns false.
func prompt(format string, a ...any) bool {
	fmt.Printf(format+" Continue? (y/n) ", a...)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				logger.Errorf("Failed to read user input: %v", err)
			}
			return false
		}
		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "y", "yes", "yeah", "yup":
			return true
		case "n", "no", "nope", "nah":
			return false
		default:
			fmt.Print("Please answer y or n: ")
		}
	}
}

// procDir recursively processes a directory and removes stale files.
// It takes the file path (fp) of the directory to process, the root folder (rootFolder) for reference,
// the age (age) in days to determine staleness, and the file extension (ext) to filter files.
// It returns a joined error of all failures encountered while processing the
// directory tree so partial cleanup is visible to the caller. The function
// continues walking siblings after an error so that every failure is reported
// rather than only the first one.
func procDir(fp, rootFolder string, age int, ext string, dryRun, pruneEmptyDirs bool) error {
	di, err := os.Stat(fp)
	if err != nil {
		return err
	}

	infos, err := getDirectoryContents(fp)
	if err != nil {
		return err
	}

	var errs []error
	for _, item := range infos {
		if item.IsDir() {
			if err := procDir(path.Join(fp, item.Name()), rootFolder, age, ext, dryRun, pruneEmptyDirs); err != nil {
				errs = append(errs, err)
			}
		} else if err := processFile(item, fp, rootFolder, age, ext, dryRun); err != nil {
			errs = append(errs, err)
		}
	}

	if err := handleEmptyDirectory(fp, di, age, ext, rootFolder, dryRun, pruneEmptyDirs); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// getDirectoryContents retrieves the contents of a directory.
func getDirectoryContents(fp string) ([]fs.FileInfo, error) {
	contents, err := os.ReadDir(fp)
	if err != nil {
		return nil, err
	}
	infos := make([]fs.FileInfo, 0, len(contents))
	for _, entry := range contents {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// processFile processes a file to determine if it should be removed.
// It returns any error from the underlying removal so the caller can
// aggregate failures across the directory tree.
func processFile(item fs.FileInfo, fp, rootFolder string, age int, ext string, dryRun bool) error {
	if isStale(item, age) && matchExt(item.Name(), ext) {
		return removeItem(path.Join(fp, item.Name()), rootFolder, dryRun)
	}
	return nil
}

// handleEmptyDirectory handles the removal of an empty directory if it is stale.
// It returns any error from the underlying removal so the caller can
// aggregate failures across the directory tree.
func handleEmptyDirectory(fp string, di fs.FileInfo, age int, ext, rootFolder string, dryRun, pruneEmptyDirs bool) error {
	empty, err := isEmpty(fp)
	if err != nil {
		return err
	}
	if empty {
		// Remove if pruneEmptyDirs is set, OR if the directory is stale and no extension filter is active.
		// The extension filter makes directory removal conservative by default to avoid deleting
		// directory structures when only certain file types are being cleaned up.
		if pruneEmptyDirs || (isStale(di, age) && ext == "") {
			return removeItem(fp, rootFolder, dryRun)
		}
	}
	return nil
}

// isEmpty checks if a directory is empty.
// It returns true if the directory is empty, false otherwise.
// An error is returned if there was a problem opening or reading the directory.
func isEmpty(name string) (empty bool, err error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

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
// It returns the error from os.Remove when removal fails so callers can
// aggregate and surface partial-failure to the user. The error is also
// logged via the existing logger.Errorf to preserve the current log format.
func removeItem(fp, rootFolder string, dryRun bool) error {
	if fp == rootFolder {
		logger.Infof("Not removing folder '%v' as it is the root folder...\n", filepath.FromSlash(fp))
		return nil
	}
	if dryRun {
		logger.Infof("[DRY RUN] '%v' would be removed...", filepath.FromSlash(fp))
		return nil
	}
	logger.Infof("Removing '%v'...", filepath.FromSlash(fp))
	if err := os.Remove(fp); err != nil {
		logger.Errorf("%v", err)
		return err
	}
	return nil
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
