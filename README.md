# rmstale

rmstale is a tool to remove stale files recursively below a given directory.  
Files and folders older than a defined period are removed.

Some examples for use:  
* Set on a schedule to clear old files from your temporary directories.
* Set on a schedule to remove downloaded files from your downloads directory.

## Install instructions

To install rmstale:  
1. From the [GitHub releases page](https://github.com/danstis/rmstale/releases) download the latest binary for your operating system.
1. Place the downloaded file into your desired location.

## Usage instructions

* rmstale is designed to be used as a command line tool, to see the latest usage instructions type `rmstale` at the command prompt:

### Command line flags:

| Flag | Description |
| -- | -- |
| -age | Period in days before an item is considered stale |
| -path | Path to a folder to process |
| -y | Allows for processing without confirmation propmt, useful for scheduling |
| -version | Displays the version of rmstale that is currently running |

### Usage examples:

```
>: rmstale -version

rmstale v1.0.0
```

```
>: rmstale -age 14 -path c:\temp
WARNING: Will remove files and folders recursively below 'c:\temp' older than 14 days. Continue?: y

-Removing 'C:\Temp\amc2E40.tmp.LOG1'...
-Removing 'C:\Temp\amc2E40.tmp.LOG2'...
-Removing 'C:\Temp\amc306D.tmp.LOG1'...
-Removing 'C:\Temp\amc306D.tmp.LOG2'...
-Removing 'C:\Temp\amc308D.tmp.LOG1'...
```

## GitHub project

Feedback, Issues, Bugs and Contribution to this tool are welcome.  
For Bugs/Issues/Feature requests, please create an issue on the [GitHub issues page](https://github.com/danstis/rmstale/issues). 

Want to contribute? Great: 
* Fork the repo using the Fork button at the top right of the GitHub repo.
* Clone the repo to your development machine, note the dependencies for this project are as follows:
	* Go version 1.8 or above
	* Go dep for dependency management - https://github.com/golang/dep
* Run `dep ensure` to create the vendor directory containing the project dependencies.
* Create a new branch for the feature that you want to contribute.
* Develop your new feature as you see fit.
* Once you have a working copy of your code, create a pull request against this project.
