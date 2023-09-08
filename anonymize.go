package anonymize_scans

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/suyashkumar/dicom"
)

// searches through the provided folder and gives all the filepaths as a slice.
func GetFilePathsInFolders(directoryPath string) ([]string, error) {
	logFileName := "GetFilePathsInSubfolders.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for GetFilePathsInSubfolders", err)
		return nil, err
	}
	defer logFile.Close()
	logger.Println("Searching through path:", directoryPath)
	var filePaths []string
	// Walk through the directory and its subdirectories
	err = filepath.WalkDir(directoryPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Check if the entry is a regular file, if not a dir add to the list.
		if !d.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	for _, paths := range filePaths {
		logger.Println(paths)
	}
	return filePaths, nil
}

// checks to see if the filepath provided is a dicom file if so return meta data.
func DicomInfoGrabber(dicomFilePath string) (map[string]string, error) {
	logFileName := "DicomInfoGrabber.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for DicomInfoGrabber:", err)
		return nil, err
	}
	defer logFile.Close()
	logger.Println("checking if :", dicomFilePath, " is a valid Dicom..")
	dicomInfo := make(map[string]string)
	dataset, err := dicom.ParseFile(dicomFilePath, nil)
	if err != nil {
		logger.Println("Error parsing :", dicomFilePath)
		return nil, err
	}
	for iter := dataset.FlatStatefulIterator(); iter.HasNext(); {
		element := iter.Next()
		dicomInfo[element.Tag.String()] = element.Value.String()
		logger.Println(element.Tag.String(), " = ", element.Value.String())
	}
	logger.Println(dicomFilePath, " is a valid dicom")
	return dicomInfo, nil
}

// searches the filepaths and if dicom file exist make note of the parent directory and return the parent directory.
func GetDicomFolders(homeDirectory, filePath string) {
	logFileName := "VerifyDicomFolder.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for DicomInfoGrabber:", err)
		return
	}
	defer logFile.Close()
	logger.Println("checking if for folders that contain valid dicoms..")
	//check if filePath given is a dicom
	dicomInfo, err := DicomInfoGrabber(filePath)
	if err != nil {
		logger.Println("Error parsing or File is not a dicom..", err)
	}
	//if the file is a dicom
	if dicomInfo != nil {
		dicomFolder := strings.TrimPrefix(filePath, homeDirectory)
		dicomFolder = strings.TrimLeft(dicomFolder, "/") //removes the leading /
		logger.Println(dicomFolder)
	}

}

// creates a logger for the functions. generates a text file and logs all the output to the text file.
func createLogger(logFileName string) (*log.Logger, *os.File, error) {
	// Create or open the log file
	//logFile, err := os.Create(logFileName)
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return nil, nil, err
	}
	// Create a logger that writes to the log file
	logger := log.New(logFile, "", log.LstdFlags)
	// Redirect standard output to the logger
	log.SetOutput(logFile)
	return logger, logFile, nil
}
