package anonymizescans

import (
	"io/fs"
	"path/filepath"
)

// grabs a directory as a type string and walks thru each file/folder given in that path.
func GetFilePathsInSubfolders(directoryPath string) ([]string, error) {
	var filePaths []string

	// Walk through the directory and its subdirectories
	err := filepath.WalkDir(directoryPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if the entry is a regular file
		if !d.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return filePaths, nil
}
