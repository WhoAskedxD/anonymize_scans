package anonymize_scans

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/suyashkumar/dicom"
)

// takes in the parent folder as a string and the content of that folder as a map, checks what type of scan it is and returns a string with the name.
// name = last set of number from folder + types of scans + FOV if CT scan is available
func MakeFolderName(parentFolder string, scanContent map[string]string) (string, error) {
	startTime := time.Now()
	//creates a logger for log files.
	logFileName := "logs/MakeFolderName.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for MakeFolderName:", err)
		return "error", err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("attempting to generate a folder name from %s", parentFolder)
	//setting folder name result
	var folderName string
	var dicomFile string
	//check to see if a CT Scan exist if so grab the Manufacture Model name and FOV
	ctPath, ctExists := scanContent["CT"]
	if ctExists {
		//grab manufacture mode name and fox
		logger.Printf("CT scan exist for path:%s", ctPath)
		dicomFiles, err := os.ReadDir(ctPath)
		if err != nil {
			fmt.Println("Error reading the directory:", err)
			return "error", err
		}
		//loop thru directory ignoring .DS_Store when it occurs and break out of loop once a dicomFile has been found.
		for _, dicom := range dicomFiles {
			if dicom.Name() != ".DS_Store" {
				dicomFile = filepath.Join(ctPath, dicom.Name())
				break
			}
		}
		logger.Printf("Checking %s", dicomFile)
		//grab values from ct scan here
	} else {
		//loop thru keys and add it to the folderName
		logger.Print("No CT scan detected running loop")
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	//fmt.Printf("\nScan Info is \n%s\n", scanInfo)
	logger.Printf("folderName is %s", folderName)
	fmt.Printf("Elapsed time: %.2f seconds for MakeFolderName\n", elapsedTime.Seconds())
	return "test", nil
}

// searches the directory given(searchFolder) and checks if the subfolders are dicom scans or not.If subfolders is a valid DicomFolderStructure add it to the []dicomFolder.
// dicomFolder is a map with the key being the parent dir path and the value being the folder info
// example:
// /Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344261765:map[
// CT:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344261765/1.2.392.200036.9163.41.127414021.344261765.12000.1
// PANO:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344261765/1.2.392.200036.9163.41.127414021.344261765.10632.1
// PARENT_FOLDER:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344261765]
func GetDicomFolders(searchFolder string) (map[string]map[string]string, error) {
	startTime := time.Now()
	//creates a logger for log files.
	logFileName := "logs/GetDicomFolders.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for GetDicomFolders:", err)
		return nil, err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("checking if %s contain valid dicom Folders", searchFolder)
	//gets a list of all the folders in the searchFolder directory
	folderList, err := ListDirectories(searchFolder)
	if err != nil {
		logger.Println("error getting list of folders:", err)
		return nil, err
	}
	//check to see if the folders in the list are scans
	dicomFolders := make(map[string]map[string]string)
	for _, folder := range folderList {
		logger.Println("checking if:", folder, "is a valid dicom folder")
		folderInfo, err := CheckDicomFolder(folder)
		if err != nil {
			logger.Println("error grabbing folderInfo", err)
		}
		//if folderinfo length is greater than 1 that means there is a scan
		if len(folderInfo) > 1 {
			logger.Println(folderInfo["PARENT_FOLDER"], "is a valid dicom folder.")
			dicomFolders[folderInfo["PARENT_FOLDER"]] = folderInfo
		}
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	//fmt.Printf("\nScan Info is \n%s\n", scanInfo)
	fmt.Printf("Elapsed time: %.2f seconds for GetDicomFolders\n", elapsedTime.Seconds())
	return dicomFolders, nil
}

// takes a parent dicomFolderPath with subfolders and returns a map with the key PARENT_FOLDER and value PATH as well as key of Scan type and value of subfolder
// example map[
// CT:/Users/harrymbp/Developer/Projects/PreXion/temp/2313920.1194420868.1125777922.144317013718248927734763.2419061/2313920.1194420868.1125777922.25137155812096533.241906.2572.11
// PANO:/Users/harrymbp/Developer/Projects/PreXion/temp/2313920.1194420868.1125777922.144317013718248927734763.2419061/2313920.1194420868.1125777922.25143587972146613.241906.5628.11
// PARENT_FOLDER:/Users/harrymbp/Developer/Projects/PreXion/temp/2313920.1194420868.1125777922.144317013718248927734763.2419061
// Scene:/Users/harrymbp/Developer/Projects/PreXion/temp/2313920.1194420868.1125777922.144317013718248927734763.2419061/2313920.1194420868.1125777922.206729801520575063.241906.824.11
// ]
func CheckDicomFolder(dicomFolderPath string) (map[string]string, error) {
	startTime := time.Now()
	//creates a logger for log files.
	logFileName := "logs/CheckDicomFolder.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for CheckDicomFolder:", err)
		// return "Error making log file for CheckDicomFolder:", err
		return nil, err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("checking to see if %s is a regular CT Scan or others", dicomFolderPath)
	//grab the subfolders in the parent folder to check which sub folder is which type of scan
	subFolderList, err := ListDirectories(dicomFolderPath)
	if err != nil {
		fmt.Println("Error getting subfolderList CheckDicomFolder:", err)
		// return "Error making log file for CheckDicomFolder:", err
		return nil, err
	}
	logger.Printf("Found subFolders\n%s in %s\n", subFolderList, dicomFolderPath)
	//make a map for the Parent and sub folder info
	folderInfo := make(map[string]string)
	folderInfo["PARENT_FOLDER"] = dicomFolderPath
	for _, subFolder := range subFolderList {
		logger.Printf("subFolder checking %s\n", subFolder)
		folderFiles, err := GetFilePathsInFolders(subFolder)
		if err != nil {
			logger.Println("error grabbing folderFiles", err)
			return nil, err
		}
		previousScanType := "NA"
		for _, file := range folderFiles {
			currentScanType, err := CheckScanType(file)
			if err != nil {
				//fmt.Println("ran into an issue checking scan type for :", file)
				continue // Skip the rest of the loop and move to the next iteration
			}
			path := filepath.Dir(file) // Remove the last part of the path and returns the directory
			logger.Printf("current scan type for %s is %s", path, currentScanType)
			if currentScanType == previousScanType {
				logger.Printf("Current Scan type is the same as the last possibly in a CT Scan breaking out of %s", path)
				break
			}
			folderInfo[currentScanType] = path
			previousScanType = currentScanType
		}
	}
	logger.Println("folderInfo is :\n", folderInfo)
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	fmt.Printf("Elapsed time: %.2f seconds for CheckDicomFolder on %s\n", elapsedTime.Seconds(), dicomFolderPath)
	return folderInfo, nil
}

// takes a dicomFile and checks to see what type of scan it is returns a string of either NA|CT|PANO|CEPH
func CheckScanType(dicomFilePath string) (string, error) {
	//creates a logger for log files.
	logFileName := "logs/CheckScanType.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for CheckScanType:", err)
		return "Error making log file for CheckScanType:", err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("checking to see what type of scan %s is", dicomFilePath)
	//check to see if file path given is a valid dicom
	dicomInfo, err := DicomInfoGrabber(dicomFilePath)
	if err != nil {
		logger.Printf("%s is not a valid dicom file", dicomFilePath)
		return "Invalid dicom file", err
	}
	//set default value for ScanType in the event 1 cannot be determined
	scanType := "NA"
	//if file is a valid dicom check this SOPClassUID
	if dicomInfo != nil {
		logger.Printf("reading %s", dicomFilePath)
		SOPClassUID := "(0008,0016)"
		//referrence
		//"1.2.840.10008.5.1.4.1.1.7" [ Secondary Capture Image Storage ] - Possibly pano Need to check Image Type as well
		//"1.2.840.10008.5.1.4.1.1.1.1" [ Digital X-Ray Image Storage - For Presentation ] - Ceph
		//"1.2.840.10008.5.1.4.1.1.2" [ CT Image Storage ] - Regular CT Scan
		//Value grabs the value from the map given the key, and found returns a boolean if key exist found will be true
		value, found := dicomInfo[SOPClassUID]
		if found {
			switch value {
			case "[1.2.840.10008.5.1.4.1.1.7]":
				logger.Printf("SOPClassUID is: %s, Possibly Pano or Saved Scene", value)
				ImageType := "(0008,0008)"
				//ImageType := "(0008,0008)"
				//referrence
				//"[ORIGINAL PRIMARY AXIAL]" - Regular CT Scan
				//"[ORIGINAL PRIMARY ]" - Pano or Ceph Scans.
				//"[DERIVED SECONDARY TERARECON]" - Saved Scene.
				image, found := dicomInfo[ImageType]
				if found {
					switch image {
					case "[ORIGINAL PRIMARY ]": //pano
						logger.Printf("Scan is a %s %s", scanType, image)
						scanType = "PANO"
					case "[ORIGINAL SECONDARY SINGLEPLANE]": //pano from eclipse
						logger.Printf("Scan is a %s %s", scanType, image)
						scanType = "PANO"
					case "[DERIVED SECONDARY TERARECON]": //scene
						scanType = "Scene"
						logger.Printf("Scan is a %s %s", scanType, image)
					default:
						logger.Printf("Scan mode not found. ImageType is :%s", image)
					}
				}
			case "[1.2.840.10008.5.1.4.1.1.1.1]":
				scanType = "CEPH"
				logger.Printf("SOPClassUID is: %s %s scan", value, scanType)
			case "[1.2.840.10008.5.1.4.1.1.2]":
				scanType = "CT"
				logger.Printf("SOPClassUID is: %s %s Scan", value, scanType)
			default:
				logger.Printf("Scan mode not found. SOPClassUID is :%s", value)
			}
		} else {
			logger.Printf("key %s not found in the map..", SOPClassUID)
		}
	}
	return scanType, nil
}

// checks to see if the filepath provided is a dicom file if so return dicom info.
func DicomInfoGrabber(dicomFilePath string) (map[string]string, error) {
	logFileName := "logs/DicomInfoGrabber.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for DicomInfoGrabber:", err)
		return nil, err
	}
	defer logFile.Close()
	//start of script
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

// creates a logger for the functions. generates a text file and logs all the output to the text file.
func createLogger(logFileName string) (*log.Logger, *os.File, error) {
	// Create or open the log file
	// Get the directory of the log file
	logDir := filepath.Dir(logFileName)

	// Create the log directory if it doesn't exist
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
			return nil, nil, err
		}
	}
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

// takes a file path and checks to see if the directory is empty if not return a slice of folder paths
func ListDirectories(folderPath string) ([]string, error) {
	// Open the directory
	dir, err := os.Open(folderPath)
	if err != nil {
		// fmt.Println("Error:", err)
		return nil, err
	}
	defer dir.Close()
	// Read the directory contents
	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		// fmt.Println("Error:", err)
		return nil, err
	}
	// Iterate over the file infos and filter out directories
	var directories []string
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			dirPath := filepath.Join(folderPath, fileInfo.Name()) // Get full directory path
			if !isEmptyDirectory(dirPath) {
				directories = append(directories, dirPath)
			}
		}
	}
	return directories, nil
}

// Function to check if a directory is empty
func isEmptyDirectory(dirPath string) bool {
	// fmt.Println("checking for empty directories", dirPath)
	dir, err := os.Open(dirPath)
	if err != nil {
		return false
	}
	defer dir.Close()

	_, err = dir.Readdir(1) // Try to read a single file
	return err != nil       //checks to see if err !=nil if it does then return value true
}

// searches through the provided folder and gives all the filepaths as a slice.
func GetFilePathsInFolders(directoryPath string) ([]string, error) {
	logFileName := "logs/GetFilePathsInSubfolders.txt"
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
