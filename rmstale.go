package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

func main() {
	path := flag.String("path", os.TempDir(), "Path to check for stale files.")
	age := flag.Int("age", 0, "Age in days to check for file modification.")
	confirm := flag.Bool("y", false, "Don't prompt for confirmation.")

	flag.Parse()

	if *age == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !*confirm {
		fmt.Printf("WARNING: Will remove files and folders recursively below %q older than %v days. Continue? (yes/no):", *path, *age)
		if !askForConfirmation() {
			fmt.Println("Operation not confirmed, exiting.")
			os.Exit(1)
		}
	}
	if err := removeFiles(*path, *age); err != nil {
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
		fmt.Println("Please type yes or no and then press enter:")
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

func removeFiles(fp string, age int) error {
	items, err := ioutil.ReadDir(fp)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.Mode().IsDir() {
			if err := removeFiles(path.Join(fp, item.Name()), age); err != nil {
				return err
			}
		} else {
			if item.ModTime().Before(time.Now().AddDate(0, 0, (age * -1))) {
				fmt.Printf("Deleting %q which is old\n", path.Join(fp, item.Name()))
			}
		}
	}
	return nil
}
