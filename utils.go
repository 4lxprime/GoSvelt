package gosvelt

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copy an file to another file
//
// CopyFile take an input file and an output file
func copyFile(inFile, outFile string) error {
	file, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer file.Close()

	newFile, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		return fmt.Errorf("utils: cannot copy %s to %s", inFile, outFile)
	}

	return nil
}

// copy directory to another directory
//
// CopyDir take an input dir and an output dir
func copyDir(srcDir, destDir string) error {
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, path[len(srcDir):])

		if info.IsDir() {
			err = os.MkdirAll(destPath, info.Mode())
			if err != nil {
				return err
			}
			return nil
		}

		err = copyFile(path, destPath)
		if err != nil {
			return fmt.Errorf("utils: cannot copy dir %s to %s", srcDir, destDir)
		}

		return nil
	})

	return err
}

// this will clean an directory
//
// for real, this will remove the dir
// and recreate an new
func cleanDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}

	return os.MkdirAll(dir, 0755)
}

// get the file of an path
// like:
// /path/to/filename.svelte -> filename.svelte
func file(file string) string {
	fileS := strings.Split(file, "/")

	return fileS[len(fileS)-1]
}

// get the filename of an path
// like:
// /path/to/filename.svelte -> filename
func fileName(file string) string {
	fileS := strings.Split(file, "/")
	fileF := fileS[len(fileS)-1]

	return strings.Split(fileF, ".")[0]
}

// This method takes a file path as input
// and returns a boolean value indicating whether
// the path represents a file or not, as well
// as an error if any.
func isFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return !fi.IsDir(), nil
}

// this will check if path exist
func exist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// write text that will be cleaned when chan is closed
func temporaryText(textChan chan struct{}, msg string) {
	fmt.Print(msg)

	select {
	case <-textChan:
		fmt.Print("\r")
		for i := 0; i < len(msg); i++ {
			fmt.Print(" ")
		}
		fmt.Print("\r")
	}
}
