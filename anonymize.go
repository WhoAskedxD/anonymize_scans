package anonymize_scans

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/suyashkumar/dicom"
)

// takes in a map of map[string]map[string]string from GetDicomFolders, and returns a list of orginal parent folder and the type of scan/possible new scan name
func GetScanNames(dicomFolders map[string]map[string]string) (map[string]string, error) {
	startTime := time.Now()
	//creates a logger for log files.
	logFileName := "logs/GetScanNames.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for GetScanName:", err)
		return nil, err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("Attempting to grab scan details")
	//create a map to store the dicom directory and scan name/details
	dicomReference := make(map[string]string)
	//loop thru the map provided to grab the parent directory and type of scan then assign it to dicomReference
	for parentFolder, scanContent := range dicomFolders {
		// fmt.Printf("Key: %s, Value: %s\n", parentFolder, scanContent)
		newName, err := GetScanName(parentFolder, scanContent)
		if err != nil {
			log.Fatal(err)
		}
		//logger.Printf("For Folder: %s, Scan Detail is: %s\n", parentFolder, newName)
		logger.Printf("%s, %s\n", parentFolder, newName)
		dicomReference[parentFolder] = newName
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	fmt.Printf("Elapsed time: %.2f seconds for GetScanNames on \n", elapsedTime.Seconds())
	return dicomReference, nil
}

// takes in the parent folder as a string and the content of that folder as a map, checks what type of scan it is and returns a string with the name.
// name = last set of number from folder + types of scans + FOV if CT scan is available example output [PreXion3D Explorer]_PANO+CT+15X15
func GetScanName(parentFolder string, scanContent map[string]string) (string, error) {
	startTime := time.Now()
	//creates a logger for log files.
	logFileName := "logs/GetScanName.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for GetScanName:", err)
		return "error", err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("attempting to generate a folder name from %s", parentFolder)
	//setting folder name result
	var folderName string
	var modelName string
	var fov string
	//dicom file to be checked for values
	var dicomFile string
	var scanTypes []string
	//SOPUID
	ManufacturerModelName := "(0008,1090)"
	for key := range scanContent {
		//logger.Printf("current key:%s", key)
		if key != "PARENT_FOLDER" {
			scanTypes = append(scanTypes, key)
		}
	}
	logger.Printf("Available scanTypes are ,%s", scanTypes)
	//check to see if a CT Scan exist if so grab the Manufacture Model name and FOV
	ctPath, ctExists := scanContent["CT"]
	if ctExists {
		//grab manufacture mode name and fox
		logger.Printf("CT scan exist for path:%s", ctPath)
		dicomFiles, err := os.ReadDir(ctPath)
		if err != nil {
			//fmt.Println("Error reading the directory:", err)
			logger.Println("Error reading the directory:", err)
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
		dicomInfo, err := DicomInfoGrabber(dicomFile)
		if err != nil {
			logger.Printf("%s is not a valid dicom file", dicomFile)
			return "Invalid dicom file", err
		}
		//if file provided is a valid dicom file grab the ManufactureModelName
		if dicomInfo != nil {
			value, found := dicomInfo[ManufacturerModelName]
			if found {
				logger.Printf("Current Model Name is %s", value)
				modelName = value
			} else {
				logger.Printf("unable to find Model name!")
			}
			//Since this is a CT Scan we can grab FOV
			//ImagePositionPatient is the (X, Y, Z) cords
			ImagePositionPatient := "(0020,0032)"
			fovValue, fovFound := dicomInfo[ImagePositionPatient]
			if fovFound {
				//do the math to get the fov of the scan FOV = (X * 2) + (Z * 2)
				//remove the brackets from the string
				fovStr := strings.Trim(fovValue, "[]")
				logger.Printf("fov values are currently %s", fovValue)
				//parse the string by spaces and add it to a slice
				fovSlice := strings.Split(fovStr, " ")
				//set the slice to a type of int
				var fovValues []int
				//go thru each element in the slice and convert the string to a float then convert the float to an int and make sure the value is always positive
				for _, valStr := range fovSlice {
					floatValue, err := strconv.ParseFloat(valStr, 64)
					if err != nil {
						fmt.Printf("Error parsing value: %v\n", err)
						return "error parsing into float", err
					}
					intValue := int(floatValue)
					//if intValue is negative turn it positive
					if intValue < 0 {
						intValue = -intValue
					}
					fovValues = append(fovValues, intValue)
				}
				//grab fov values multiple times 2 and then divide by 10 to grab the first 2 digits, then convert it to a string.
				xValue := strconv.Itoa((fovValues[0] * 2) / 10)
				yValue := strconv.Itoa((fovValues[2] * 2) / 10)
				//set fov string
				fov = "+" + xValue + "X" + yValue
				logger.Printf("Xvalue is currently:%s YValue is currently:%s", xValue, yValue)

			} else {
				logger.Printf("unable to find FOV!")
			}

		}
	} else {
		//grab a dicomfile from the list of available scan types
		logger.Printf("selecting scan type %s to grab info", scanTypes[0])
		//check to see if the scanType is present if so grab value
		dicomPath, dicomExist := scanContent[scanTypes[0]]
		if dicomExist {
			logger.Printf("Searching %s at path %s", scanTypes[0], dicomPath)
			dicomFiles, err := os.ReadDir(dicomPath)
			if err != nil {
				fmt.Println("Error reading the directory:", err)
				return "error", err
			}
			//loop thru directory ignoring .DS_Store when it occurs and break out of loop once a dicomFile has been found.
			for _, dicom := range dicomFiles {
				if dicom.Name() != ".DS_Store" {
					dicomFile = filepath.Join(dicomPath, dicom.Name())
					break
				}
			}
			logger.Printf("Checking %s", dicomFile)
			dicomInfo, err := DicomInfoGrabber(dicomFile)
			if err != nil {
				logger.Printf("%s is not a valid dicom file", dicomFile)
				return "Invalid dicom file", err
			}
			//if file provided is a valid dicom file grab the ManufactureModelName
			if dicomInfo != nil {
				value, found := dicomInfo[ManufacturerModelName]
				if found {
					logger.Printf("Current Model Name is %s", value)
					modelName = value
				}
			}
		} else {
			logger.Printf("unable to locate DicomPath!!")
		}

	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	//fmt.Printf("\nScan Info is \n%s\n", scanInfo)
	//construct the Folder Name
	listOfScans := strings.Join(scanTypes, "+")
	folderName = modelName + "_" + listOfScans + fov
	logger.Printf("folderName is %s", folderName)
	fmt.Printf("Elapsed time: %.2f seconds for GetScanName on %s\n", elapsedTime.Seconds(), parentFolder)
	return folderName, nil
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
		logger.Println("checking :", folder)
		folderInfo, err := CheckDicomFolder(folder)
		if err != nil {
			logger.Println("error grabbing folderInfo", err)
		}
		//if folderinfo length is greater than 1 that means there is a scan
		if len(folderInfo) >= 1 {
			logger.Println(folder, "is a valid dicom folder.")
			dicomFolders[folder] = folderInfo
		}
	}
	for key, value := range dicomFolders {
		logger.Printf("folder info is %s\n%s\n\n", key, value)
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	//fmt.Printf("\nScan Info is \n%s\n", scanInfo)
	fmt.Printf("Elapsed time: %.2f seconds for GetDicomFolders searching thru %s\n", elapsedTime.Seconds(), searchFolder)
	return dicomFolders, nil
}

// takes a parent dicomFolderPath with subfolders and returns a map with the key PARENT_FOLDER and value PATH as well as key of Scan type and value of subfolder
// example map[
// ManufacturerModelName:[PreXion3D Explorer]
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
	logger.Printf("Found %d subFolders\n%s in %s\n", len(subFolderList), subFolderList, dicomFolderPath)
	//make a map for the Parent and sub folder info
	folderInfo := make(map[string]string)
	// folderInfo["PARENT_FOLDER"] = dicomFolderPath
	for index, subFolder := range subFolderList {
		logger.Printf("subFolder checking %s\n", subFolder)
		//entered subfolder grabbing files to loop thru
		folderFiles, err := GetFilePathsInFolders(subFolder)
		if err != nil {
			logger.Println("error grabbing folderFiles", err)
			return nil, err
		}
		//grab a file from the subfolder and check to see if file is a valid dicom file if so assign the value and break out of the loop
		for _, file := range folderFiles {
			currentFileInfo, err := CheckScanType(file)
			//if the file is not a valid dicom file continue(skip) current file and ignore the rest of the loops function with the "continue"
			if err != nil {
				//fmt.Println("ran into an issue checking scan type for :", file)
				continue // Skip the rest of the loop and move to the next iteration
			}
			logger.Printf("Looking at %s it currently contains\n%s", file, currentFileInfo)
			logger.Printf("current folder info is :%s", folderInfo)
			for currentKey, currentValue := range currentFileInfo {
				value, found := folderInfo[currentKey]
				if found && value == currentValue {
					logger.Printf("found duplicate key:%sand value:%s pair breaking out of this loop", currentKey, value)
					continue // skip assignment for this key value pair
				} else if found && value != currentValue {
					newKey := currentKey + strconv.Itoa(index)
					logger.Printf("found matching key:%s but different values:%s, making a new key:%s", currentKey, currentValue, newKey)
					folderInfo[newKey] = currentValue
				} else {
					folderInfo[currentKey] = currentValue
					logger.Printf("assigning key:%s value:%s to %s", currentKey, currentValue, folderInfo)
				}
			}
			break
		}
	}
	logger.Printf("folderInfo is :%s\n\n\n", folderInfo)
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	fmt.Printf("Elapsed time: %.2f seconds for CheckDicomFolder on %s\n", elapsedTime.Seconds(), dicomFolderPath)
	return folderInfo, nil
}

// takes a dicomFile and checks to see what type of scan it is returns a string of either NA|CT|PANO|CEPH
func CheckScanType(dicomFilePath string) (map[string]string, error) {
	//creates a logger for log files.
	logFileName := "logs/CheckScanType.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for CheckScanType:", err)
		return nil, err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("checking to see what type of scan %s is", dicomFilePath)
	//check to see if file path given is a valid dicom
	dicomInfo, err := DicomInfoGrabber(dicomFilePath)
	if err != nil {
		logger.Printf("%s is not a valid dicom file", dicomFilePath)
		return nil, err
	}
	// Set default value for ScanType in the event 1 cannot be determined
	scanType := "NA"
	// Remove the last part of the file path to give the directory
	path := filepath.Dir(dicomFilePath)
	//make a map to store the information
	dicomContents := make(map[string]string)
	//if file is a valid dicom check this SOPClassUID
	if dicomInfo != nil {
		SOPClassUID := "(0008,0016)"
		ManufacturerModelName := "(0008,1090)"
		//referrence
		//"1.2.840.10008.5.1.4.1.1.7" [ Secondary Capture Image Storage ] - Possibly pano Need to check Image Type as well
		//"1.2.840.10008.5.1.4.1.1.1.1" [ Digital X-Ray Image Storage - For Presentation ] - Ceph
		//"1.2.840.10008.5.1.4.1.1.2" [ CT Image Storage ] - Regular CT Scan
		//Value grabs the value from the map given the key, and found returns a boolean if key exist found will be true
		value, found := dicomInfo[SOPClassUID]
		if found {
			logger.Printf("path is %s\n", path)
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
		//Look for ManufacturerModelName and add to dicomContents if available
		modelName, foundModel := dicomInfo[ManufacturerModelName]
		if foundModel {
			logger.Printf("found Model name %s\n", modelName)
			dicomContents["ManufacturerModelName"] = modelName
		}

	}
	dicomContents[scanType] = path
	//if there is a CT scan grab the FOV Value
	if scanType == "CT" {
		dicomContents["FOV"] = "15x15XTest"
	}
	logger.Printf("end of script content is\n%s\n", dicomContents)
	return dicomContents, nil
}

func GetFOVSize(dicomInfo map[string]string) (string, error) {
	startTime := time.Now()
	//creates a logger for log files.
	logFileName := "logs/GetFOVSize.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for GETFOVSize:", err)
		return "error", err
	}
	defer logFile.Close()
	//start of script
	logger.Printf("Starting function GetFOVSize")
	//setting varible
	// ImagePositionPatient := "(0020,0032)"

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	//construct the Folder Name
	fmt.Printf("Elapsed time: %.2f seconds for GetFOVSize\n", elapsedTime.Seconds())
	return "string", nil
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

// Block of code for logger
// startTime := time.Now()
// //creates a logger for log files.
// logFileName := "logs/GetFOVSize.txt"
// logger, logFile, err := createLogger(logFileName)
// if err != nil {
// 	fmt.Println("Error making log file for GETFOVSize:", err)
// 	return "error", err
// }
// defer logFile.Close()
// //start of script
// logger.Printf("Starting function GetFOVSize")
// main function starts here
// main function ends here
// endTime := time.Now()
// elapsedTime := endTime.Sub(startTime)
// //construct the Folder Name
// fmt.Printf("Elapsed time: %.2f seconds for GetFOVSize\n", elapsedTime.Seconds())
// return "string", nil
