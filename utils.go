package gosvelt

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cp "github.com/otiai10/copy"
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

	// writing full path
	outFileDir := filepath.Dir(outFile)
	if _, err := os.Stat(outFileDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outFileDir, 0755); err != nil {
			return err
		}
	}

	newFile, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		return fmt.Errorf("utils: cannot copy %s to %s (%s)", inFile, outFile, err)
	}

	return nil
}

// copy directory to another directory
//
// CopyDir take an input dir and an output dir
func copyDir(srcDir, destDir string) error {
	err := cp.Copy(srcDir, destDir, cp.Options{
		Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
			// todo: add some suffix
			return strings.HasSuffix(src, ".git"), nil
		},
	})
	if err != nil {
		return fmt.Errorf("utils: cannot copy dir %s to %s (%s)", srcDir, destDir, err)
	}

	return nil
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

// this will check if path fileExists
func fileExists(path string) bool {
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

func calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func calculateTreeHash(root string) (string, error) {
	var fileHashes []string

	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			hash, err := calculateFileHash(path)
			if err != nil {
				return err
			}
			fileHashes = append(fileHashes, hash)
		}

		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(fileHashes)
	hasher := sha256.New()

	if _, err := hasher.Write([]byte(strings.Join(fileHashes, ""))); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
