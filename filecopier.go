package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

/*
NOTE: This code requires that the source and destination directories both share the same structure, and that the files to be copied are in leaves; directories containing no other directories.
Feel free to adapt this code for your own use case.
*/

//The three parameters to be read in.
var dest, source, pattern *string

var deleteEmpty, deleteEmptyBefore, maintainOriginal *bool //Option to delete empty directories in the destination if they would still be empty after the copy, or keep older files

//Counters for the number of files copied, directories skipped, those with no matching file found and the deleted empty directories.
var copied, skipped, noMatch, deleted int

func main() {
	//Read in the command arguments, and parse
	dest = flag.String("dest", "", "The parent directory of the destination music. Must have the same hierarchy as the source. E.g. If your music is \"\\Music\\%Artist%\\%Album%\\%files%\", then the \"dest\" would be \"\\Music\"")
	source = flag.String("source", "", "The parent directory of the source music and artwork. Must have the same hierarchy as the destination. E.g. If your music is \"\\Music\\%Artist%\\%Album%\\%files%\", then the \"source\" would be \"\\Music\"")
	pattern = flag.String("ext", "?older.*", "The file format to scan for, formatted as golang glob. Default \"?older.*\"")
	deleteEmpty = flag.Bool("delete", false, "Deletes empty directories in the destination if it would remain empty after copying all matching files")
	deleteEmptyBefore = flag.Bool("before", false, "Deletes directories in the destination that are empty BEFORE copying")
	maintainOriginal = flag.Bool("maintain", false, "Does not overwrite files in the destination")
	flag.Parse()

	//Checks if the source and destination have been specified.
	if *source == "" {
		fmt.Println("No source folder specified. Use the \"source\" option to set, or \"-h\" for help Exiting...")
		os.Exit(1)
	} else if *dest == "" {
		fmt.Println("No destination folder specified. Use the \"dest\" option to set, or \"-h\" for help. Exiting...")
		os.Exit(1)
	}

	//Walks the directory and copies the specified file type.
	if err := filepath.Walk(*dest, visit); err != nil {
		fmt.Printf("An error occured: %v\n", err)
	}
	//Prints a summary of the process
	fmt.Printf("Copied: %d    Skipped: %d    No Match Found: %d    Deleted: %d", copied, skipped, noMatch, deleted)
}

func visit(path string, f os.FileInfo, err error) error {

	/*
		filepath.Walk() visits each file and folder within the specified directory (in this case, the destination), in sequential order.
		A further explanation of how the code works with examples can be found in the README on the GitHub page.
	*/

	//Checks if the current item is a file or folder. As we are only interested in folders, specifically the lowest level folders,
	//any file is skipped.
	if f.IsDir() {

		//The following loop checks to see if the current directory contains only files.
		files, _ := filepath.Glob(path + "/*")
		isLeaf := true
		for _, file := range files {
			isDirec, _ := os.Stat(file)
			if isDirec.IsDir() {
				isLeaf = false
			}
		}

		//If the directory contains only files, attempt to copy the specified pattern from the source to the destination
		if isLeaf {

			//Creates the full path to the source files. Performs a straight swap of the destination parent directory with the
			//source parent directory
			folderPath := strings.TrimPrefix(path, strings.Replace(*dest, "/", "\\", -1)+"\\")
			sourcePath := strings.Join([]string{strings.Replace(*source, "/", "\\", -1), folderPath}, "\\")

			//Fetches the slices of files matching the specified pattern
			sourceMatching, _ := filepath.Glob(sourcePath + "\\" + *pattern)
			//destMatching, _ := filepath.Glob(path + "\\" + *pattern)

			fileList, _ := filepath.Glob(path + "\\*")
			if *deleteEmptyBefore && len(fileList) == 0 {
				deleteDirectory(path)
			} else {

				//Checks if any files actually matched in the source
				if len(sourceMatching) > 0 {

					dirCopy := 0

					//Copies each file matching the expression if the destination does not already contain a newer version
					for _, file := range sourceMatching {

						fileDetails, _ := os.Stat(file)
						destinationPath := path + "\\" + fileDetails.Name()
						destDet, err := os.Stat(destinationPath)

						if os.IsNotExist(err) || (!*maintainOriginal && destDet.ModTime().Before(fileDetails.ModTime())) {
							//Opens the source file to copy
							from, _ := os.Open(file)
							defer from.Close()

							//Opens the destination file to copy to, creating one if it doesn't exist
							to, _ := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE, 0666)
							defer to.Close()

							io.Copy(to, from)
							copied++
							dirCopy++
						}
					}
					if dirCopy > 0 {
						fmt.Printf("Copied %d file(s): %s\n", dirCopy, path)

					} else {
						if *deleteEmpty {
							if err := deleteDirectory(path); err != nil {
								fmt.Println("Already contains matching file(s): ", path)
								skipped++
							}
						} else {
							fmt.Println("Already contains matching file(s): ", path)
							skipped++
						}

					}
				} else {
					if *deleteEmpty {
						if err := deleteDirectory(path); err != nil {
							fmt.Printf("No matching file(s) found: %s\n", path)
							noMatch++
						}
					} else {
						fmt.Printf("No matching file(s) found: %s\n", path)
						noMatch++
					}
				}
			}
		}

	}

	return nil
}

//deleteDirectory deletes the specified directory and it's parents if empty, returning an error if not
func deleteDirectory(path string) error {
	//Verifies a directory not a file has been passed through
	dir, _ := os.Stat(path)
	if dir.IsDir() {
		parent := path[:strings.LastIndex(path, "\\")]

		//This assumes the path passed to the function is a child of the destination directory
		if path == *dest {
			return nil
		} else {
			//os.Remove() will produce an error if the directory is not empty instead of removing it
			if err := os.Remove(path); err != nil {
				return err
			} else {
				fmt.Printf("Deleted empty folder: %s\n", path)
				deleted++
				deleteDirectory(parent)
				return nil
			}
		}
	} else {
		return errors.New("Specified path is not a directory")
	}
}
