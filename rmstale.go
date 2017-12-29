package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// AppVersion controls the application version number
var AppVersion string = "0.0.0"
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
		fmt.Printf("WARNING: Will remove files and folders recursively below %q older than %v days. Continue? (y/n):", folder, age)
		if !askForConfirmation() {
			fmt.Println("Operation not confirmed, exiting.")
			os.Exit(1)
		}
	}
	if err := filepath.Walk(folder, walkfunc); err != nil {
		fmt.Printf("failed to read directory: %v\n", err)
		os.Exit(1)
	}

	// Run a second time to remove empty folders
	if err := filepath.Walk(folder, walkfunc); err != nil {
		fmt.Printf("failed to read directory: %v\n", err)
		os.Exit(1)
	}
}

// askForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type y or n and then press enter:")
		return askForConfirmation()
	}
}

// You might want to put the following two functions in a separate utility package.

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

func walkfunc(fp string, fi os.FileInfo, err error) error {
	if fi.IsDir() {
		empty, err := isEmpty(fp)
		if err != nil {
			return err
		}
		if empty && fi.ModTime().Before(time.Now().AddDate(0, 0, (age*-1))) {
			if err = removeItem(fp); err != nil {
				return err
			}
		}
	} else {
		if fi.ModTime().Before(time.Now().AddDate(0, 0, (age * -1))) {
			if err = removeItem(fp); err != nil {
				return err
			}
		}
	}
	return nil
}

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

func removeItem(fp string) error {
	if fp == folder {
		return fmt.Errorf("not removing folder %q as it is the root folder", fp)
	}
	fmt.Printf("Removing %q\n", fp)
	err := os.Remove(fp)
	return err
}
