package anonymize_scans

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// takes a filelist[]string and details, opens the dicom read and modify data then create new file at output location
func MakeDicom(fileList []string, outputPath string, newDicomAttribute map[tag.Tag]string) error {
	// Block of code for logger
	startTime := time.Now()
	// creates a logger for log files.
	logFileName := "logs/MakeDicom.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for MakeDicom:", err)
		return err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of MakeDicom Script ---------")
	// main function starts here
	logger.Printf("Output Folder path is:%s", outputPath)
	//Grabbing the SeriesInstanceUID to add an index at the end for both MediaStorageSOPInstanceUID|SOPInstanceUID

	for index, filePath := range fileList {
		// logger.Printf("opening %s", filePath)
		currentDataset, err := dicom.ParseFile(filePath, nil)
		//if file isnt a valid dicom file skip the proccess
		if err != nil {
			logger.Printf("error parsing:%s as a dicom fille copying file over instead\n", filePath)
			err := copyFile(filePath, outputPath)
			if err != nil {
				logger.Printf("error copying file")
			}
			continue
		}
		// logger.Printf("found dataset %s", currentDataset)
		// grab SeriesInstanceUID from newDicomAttribute and add the index of the loop to the end to generate a new MediaStorageSOPInstanceUID|SOPInstanceUID
		newDicomAttribute[tag.MediaStorageSOPInstanceUID] = newDicomAttribute[tag.SeriesInstanceUID] + "." + strconv.Itoa(index+1)
		newDicomAttribute[tag.SOPInstanceUID] = newDicomAttribute[tag.MediaStorageSOPInstanceUID]
		for tag, value := range newDicomAttribute {
			element, err := currentDataset.FindElementByTag(tag)
			if err != nil {
				log.Println("unable to locate ", tag, value)
				continue
			}
			// log.Println("found tag ", element.Value.String())
			newValue, _ := dicom.NewValue([]string{value}) //create a new dicom value with the interface of []string where value is the newDicomAttribute
			// log.Println("new value is ", newValue.String())
			element.Value = newValue // assign the current elements value to the newValue
		}
		//format the index so that it can make a file name 000_Index.dcm
		name := fmt.Sprintf("%04d", index)
		output_File := filepath.Join(outputPath, name+".dcm")
		// log.Println("dicom dataset modified creating new dicom file at  :", output_File)
		//create a new file with the given output_file name and path
		newDicomFile, err := os.Create(output_File)
		if err != nil {
			log.Println("error makign file!", err)
			return err
		}
		//defer means to close up once the function createDicom finishes running
		defer newDicomFile.Close()
		//write to the file at newDicomFile with the data from currentDataset
		dicom.Write(newDicomFile, currentDataset)
		// logger.Printf("Done Making dicom file:%s", output_File)
	}
	// main function ends here
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("Elapsed time: %.2f seconds for %s\n", elapsedTime.Seconds(), outputPath)
	logger.Printf("------- End of MakeDicom Script ---------\n\n")
	logger.Printf("Elapsed time: %.2f seconds for MakeDicom\n", elapsedTime.Seconds())
	return nil
}

// takes in dicomFolderPath map[string]string ([dicomfolder]outputFolder), newDicomAttribute map[tag.Tag]string| Open the file and modify it then save to output Path
func MakeDicomFolders(folderInfo map[string]string, newDicomAttribute map[tag.Tag]string) error {
	// Block of code for logger
	startTime := time.Now()

	// creates a logger for log files.
	logFileName := "logs/MakeDicomFolder.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for MakeDicomFolder:", err)
		return err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of MakeDicomFolder Script ---------")
	// main function starts here
	//generate timestamp for the StudyUID
	unformattedDate := startTime.Format("20061230")                                //this format generated 3 extra characters for some reason we need to clean it up and remove the last 3
	date := unformattedDate[:len(unformattedDate)-3]                               //removes the last 3 characters
	hour, min, sec := startTime.Clock()                                            //grabs the hour min and seconds as Ints
	timestamp := date + strconv.Itoa(hour) + strconv.Itoa(min) + strconv.Itoa(sec) //constructs the timestamp to be used as a studyuid
	logger.Printf("current timestamp is  %v", timestamp)
	//grab and set new StudyUID here StudyUID will stay the same for the parent folder
	newStudyInstanceUID := newDicomAttribute[tag.StudyInstanceUID] + "." + timestamp
	logger.Printf("current StudyInstanceUID is:%s\nnewStudyInstanceUID:%s", newDicomAttribute[tag.StudyInstanceUID], newStudyInstanceUID)
	//assign the new value to the actual StudyInstanceUID
	newDicomAttribute[tag.StudyInstanceUID] = newStudyInstanceUID
	for parentFolder, outputFolder := range folderInfo {
		//grab the StudyInstanceUID generate 5 digit random number and add it onto the end for the series instance UID| new Series generated each loop
		//setting the randomNumber for this Series
		randomNumber := rand.Intn(10000 - 1)
		//grabbing the SeriesInstanceUID and adding the randomgenerated number to the end of it.
		newDicomAttribute[tag.SeriesInstanceUID] = newDicomAttribute[tag.StudyInstanceUID] + "." + strconv.Itoa(randomNumber)
		logger.Printf("current grabing dicoms from:%s\nmodifying and outputing at:%s\n", parentFolder, outputFolder)
		//grab all the files from the parentFolder
		folderList, err := GetFilePathsInFolders(parentFolder)
		if err != nil {
			log := "error getting file list in %s"
			logger.Printf(log, parentFolder)
			return fmt.Errorf(log, parentFolder)
		}
		logger.Printf("grabbing files from:%s", parentFolder)
		//open each file from the folder list and make changes then save to output Path.
		err = MakeDicom(folderList, outputFolder, newDicomAttribute)
		if err != nil {
			log := "error making dicoms from :%s"
			logger.Printf(log, folderList)
			return fmt.Errorf(log, folderList)
		}
		logger.Printf("done making dicoms..")
	}
	// main function ends here
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("------- End of MakeDicomFolder Script ---------\n\n")
	fmt.Printf("Elapsed time: %.2f seconds for MakeDicomFolders\n", elapsedTime.Seconds())
	return nil
}

// takes in dicomFolderPath, OutputFolderPath ,UID and scanDetails, then generates a []string with the output folderPaths.
func MakeOutputPath(parentFolderPath, outputFolderPath string, uid int, scanDetail map[string]string) (map[string]string, error) {
	startTime := time.Now()
	// creates a logger for log files.
	logFileName := "logs/MakeOutputPath.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for MakeOutputPath:", err)
		return nil, err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of MakeOutputPath Script ---------")
	// main function starts here
	logger.Printf("\nparentFolderPath :%s\noutputFolderPath :%s\nUID is %d\nScanDetail :%s\n", parentFolderPath, outputFolderPath, uid, scanDetail)
	//Makes a map storing the original dicomFolder path as they key and the outputDicomFolder path as the value
	outputPaths := make(map[string]string)
	//filter the scan details out
	// scans, patientInfo, err := FilterScanDetails(scanDetails)
	//generate new Parent folder name for scan
	newParentFolderName, err := MakeScanName(scanDetail)
	//converts the UID to a string and adds it to the end of the folder name.
	newParentFolderName += "_" + strconv.Itoa(uid)
	if err != nil {
		fmt.Println("Error making log file for MakeOutputPath:", err)
		return nil, err
	}
	logger.Printf("New parentfoldername is:%s", newParentFolderName)
	//create parent folder with the new name
	outputParentPath := filepath.Join(outputFolderPath, newParentFolderName)
	logger.Printf("output parent folder path is %s", outputParentPath)
	//make parentDir for scans using outputParentPath | os.ModePerm is a constant that gives the folder full read, write, and execute permissions
	err = os.Mkdir(outputParentPath, os.ModePerm)
	// Check for errors | if folder already exist it will error out
	if err != nil {
		fmt.Println("Error creating folder:", err)
		return nil, err
	}
	logger.Printf("created Parent folder at:%s\n", outputParentPath)
	// look thru scanDetails and create a folder each scan that exist
	for scan, detail := range scanDetail {
		//check if the scan detail is a type of scan (CT|PANO|CEPH|SCENE) or just details | if key is the following ignore it (FOV|ManufacturerModelName|)
		logger.Printf("current key is:%s with detail as:%s", scan, detail)
		switch scan {
		case "ManufacturerModelName":
			logger.Printf("found %s ignoring", scan)
		case "FOV":
			logger.Printf("found %s ignoring", scan)
		case "PatientBirthDate":
			logger.Printf("found %s ignoring", scan)
		case "PatientID":
			logger.Printf("found %s ignoring", scan)
		case "PatientName":
			logger.Printf("found %s ignoring", scan)
		default:
			subfolderPath := filepath.Join(outputParentPath, scan)
			outputPaths[detail] = subfolderPath
			logger.Printf("creating a folder for %s at %s", scan, subfolderPath)
			err = os.Mkdir(subfolderPath, os.ModePerm)
			// Check for errors | if folder already exist it will error out
			if err != nil {
				fmt.Println("Error creating folder:", err)
				return nil, err
			}
			logger.Printf("created Subfolder %s for %s", subfolderPath, scan)
		}

	}
	if len(outputPaths) == 0 {
		log := "there was an issue assigning a output folder to the dicom folder %s"
		logger.Printf(log, scanDetail)
		return nil, fmt.Errorf(log, scanDetail)
	}
	//test
	// name := "/Users/harrymbp/Developer/Projects/PreXion/output/[PX3D Eclipse]+PANO_4"
	// err = os.Mkdir(name, os.ModePerm)
	// if err != nil {
	// 	fmt.Println("Error creating folder:", err)
	// }
	// main function ends here
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("output paths generated are:%s\n", outputPaths)
	logger.Printf("------- End of MakeOutputPath Script ---------\n\n")
	logger.Printf("Elapsed time: %.2f seconds for MakeOutputPath\n", elapsedTime.Seconds())
	return outputPaths, nil
}

// log which scan was modified and the new info for that scan| Keep track of the orginal Patient|ID|
func LogAnonymizedScan(scanDetails map[string]string, newScanInfo map[tag.Tag]string) (map[string]string, error) {
	// Block of code for logger
	startTime := time.Now()
	// creates a logger for log files.
	logFileName := "logs/LogAnonymizedScan.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for LogAnonymizedScan:", err)
		return nil, err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of LogAnonymizedScan Script ---------")
	// main function starts here
	logger.Printf("current scan info is %s\nnewScanInfo is %s\n", scanDetails, newScanInfo)
	//create a map to store the results
	// should contain LOCATION:Folderpath and Original Patient ID:New Patient ID
	loggedInfo := make(map[string]string)
	//list of scans to check
	scans := []string{"CT", "PANO", "SCENE", "CEPH"}
	//loop thru scan info to find a scan and grab the parent directory of the scan| loop only needs to run once until a valid scan is found then it can break
	for currentScanType, currentScanDetails := range scanDetails {
		matchFound := false
		//if a scan type is found grab the folderPath | original PatientID | newPatientID and assign them to loggedInfo
		for _, match := range scans {
			if currentScanType == match {
				logger.Printf("Current ScanType is %s\nValue is %s", currentScanType, currentScanDetails)
				folderPath := filepath.Dir(currentScanDetails) //grab the folder path for the scan
				//value assignments
				loggedInfo["LOCATION"] = folderPath
				loggedInfo["ORGINIALPATIENTID"] = scanDetails["PatientID"]
				loggedInfo["NEWPATIENTID"] = newScanInfo[tag.PatientID]
				matchFound = true

			}
		}
		if matchFound {
			logger.Printf("LOCATION is: %s and ORGINIALPATIENTID is:%s, NEWPATIENTID is:%s", loggedInfo["LOCATION"], loggedInfo["ORGINIALPATIENTID"], loggedInfo["NEWPATIENTID"])
			break
		}
	}
	if len(loggedInfo) <= 0 {
		logError := ("unable to locate any scans and grab any info")
		logger.Printf(logError)
		err = errors.New(logError)
		return nil, err
	}
	// main function ends here
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("------- End of LogAnonymizedScan Script ---------\n\n")
	logger.Printf("Elapsed time: %.2f seconds for LogAnonymizedScan\n", elapsedTime.Seconds())
	return loggedInfo, nil
}

// takes in scanDetails and generates a new PatientName|PatientID|PatientDOB returns a map[tag.Tag]string
func RandomizePatientInfo(scanDetails map[string]string) (map[tag.Tag]string, error) {
	startTime := time.Now()
	// creates a logger for log files.
	logFileName := "logs/RandomizePatientInfo.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for RandomizePatientInfo:", err)
		return nil, err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of RandomizePatientInfo Script ---------")
	//make map to hold new randomized content
	randomizedDicomAttributes := make(map[tag.Tag]string)
	//make variables to store the results to construct the final name later
	var newPatientName string
	var newPatientID string
	var newPatientBirthDate string
	for key, value := range scanDetails {
		logger.Printf("current key is: %s and the value is: %s\n", key, value)
		switch key {
		case "PatientName":
			//take in patient id and modify it to have the following syntax ->Initals of name_list of scan or scan mode ->HV_CT+PANO+15x15
			logger.Printf("%s found modifying the %s", key, value)
			//Extract the first character
			firstInital := value[1:2]
			//Find the index of '^' and extract the character after it
			indexOfCaret := strings.Index(value, "^")
			lastInital := value[indexOfCaret+1 : indexOfCaret+2]
			initals := firstInital + lastInital
			//get a list of scans and
			scans, err := MakeScanName(scanDetails)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}
			logger.Printf("Initals are %s and scans are %s", initals, scans)
			newPatientName = initals + "^" + scans
			logger.Printf("newPatientName is %s", newPatientName)
			randomizedDicomAttributes[tag.PatientName] = newPatientName

		case "PatientID":
			logger.Printf("%s found modifying the %s", key, value)
			currentTime := time.Now().Unix()
			//generate random Number
			randomNumber := rand.Intn(10000-1) + int(currentTime)
			newPatientID = strconv.Itoa(randomNumber)
			logger.Printf("newPatientId is %s", newPatientID)
			randomizedDicomAttributes[tag.PatientID] = newPatientID

		case "PatientBirthDate":
			logger.Printf("%s found modifying the %s", key, value)
			//keep the current year and set month and day to 1230 december 30th
			//check if there is a dob by checking the length of value if there is no dob the default value is "[]" which is 2 characters.
			if len(value) <= 2 {
				logger.Printf("no DOB found ignoring DOB")
			} else {
				year := value[1:5]
				newPatientBirthDate = year + "1230"
				randomizedDicomAttributes[tag.PatientBirthDate] = newPatientBirthDate
				logger.Printf("newPatientBirthDate is %s", newPatientBirthDate)
			}
		}
	}
	//construct the map and return it

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("newPatientName is:%s newPatientID is:%s newPatientBirthDate is:%s", newPatientName, newPatientID, newPatientBirthDate)
	logger.Printf("------- End of RandomizePatientInfo Script ---------\n\n")
	logger.Printf("Elapsed time: %.2f seconds for RandomizePatientInfo\n", elapsedTime.Seconds())
	return randomizedDicomAttributes, nil
}

// Takes in scanDetails and returns a list of scan types
func GetScanList(scanDetails map[string]string) ([]string, error) {
	startTime := time.Now()
	// creates a logger for log files.
	logFileName := "logs/GetScanList.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for GetScanList:", err)
		return nil, err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of GetScanList Script ---------")
	//initalizing variables for the scan name
	var ListOfScans []string
	for key, value := range scanDetails {
		logger.Printf("current key is: %s and the value is: %s\n", key, value)
		switch key {
		case "CT":
			logger.Printf("%s found adding %s to the Scans", key, value)
			ListOfScans = append(ListOfScans, key)
		case "PANO":
			logger.Printf("%s found adding %s to the Scans", key, value)
			ListOfScans = append(ListOfScans, key)
		case "CEPH":
			logger.Printf("%s found adding %s to the Scans", key, value)
			ListOfScans = append(ListOfScans, key)
		case "SCENE":
			logger.Printf("%s found adding %s to the Scans", key, value)
			ListOfScans = append(ListOfScans, key)
		}
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("amount of scans are %d\nand the list is :%s", len(ListOfScans), ListOfScans)
	logger.Printf("------- End of GetScanList Script ---------\n\n")
	logger.Printf("Elapsed time: %.2f seconds for GetScanList\n", elapsedTime.Seconds())
	return ListOfScans, nil
}

// takes in a map of scan details and constructs a string with the details
func MakeScanName(scanDetails map[string]string) (string, error) {
	startTime := time.Now()
	// creates a logger for log files.
	logFileName := "logs/MakeScanName.txt"
	logger, logFile, err := createLogger(logFileName)
	if err != nil {
		fmt.Println("Error making log file for MakeScanName:", err)
		return "error", err
	}
	defer logFile.Close()
	// start of script
	logger.Printf("------- Start of MakeScanName Script ---------")
	//initalizing variables for the scan name
	var ManufactureModelName string
	var Fov string
	var CompleteName string
	//grab a []string of scans and add them together.
	ListOfScans, err := GetScanList(scanDetails)
	if err != nil {
		log.Fatal(err)
	}
	for key, value := range scanDetails {
		logger.Printf("current key is: %s and the value is: %s\n", key, value)
		switch key {
		case "ManufacturerModelName":
			logger.Printf("%s found adding %s to the ManufacturerModelName", key, value)
			ManufactureModelName = value
		case "FOV":
			logger.Printf("%s found adding %s to the Fov", key, value)
			Fov = "+" + value
		}
	}
	//create a string from all the slices of scans
	Scans := strings.Join(ListOfScans, "+")
	//check if a name or scan type was assigned if not return an error
	if ManufactureModelName == "" || Scans == "" {
		log := "no Manufacture name or Scan type"
		logger.Printf(log)
		return log, fmt.Errorf(log)
	}
	//construct the name
	CompleteName = ManufactureModelName + "+" + Scans + Fov
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("Complete name is %s", CompleteName)
	logger.Printf("------- End of MakeScanName Script ---------\n\n")
	logger.Printf("Elapsed time: %.2f seconds for MakeScanName\n", elapsedTime.Seconds())
	return CompleteName, nil
}

// searches the directory given(searchFolder) and checks if the subfolders are dicom scans or not.If subfolders is a valid DicomFolderStructure add it to the []dicomFolder.
// example output [output paths]map[type of scan or scan detail]values
// /Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344460687
// map[CT:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344460687/1.2.392.200036.9163.41.127414021.344460687.8332.1 FOV:15X15 ManufacturerModelName:[PreXion3D Explorer] PANO:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344460687/1.2.392.200036.9163.41.127414021.344460687.11336.1 PatientBirthDate:[20010101] PatientID:[02181963] PatientName:[Case^Number 8]]
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
	dicomFolders := make(map[string]map[string]string)
	//check to see if the folders in the list are scans
	for _, folder := range folderList {
		logger.Println("checking :", folder)
		folderInfo, err := CheckDicomFolder(folder)
		//if not a valid folder skip and check the next.
		if err != nil {
			logger.Println("error grabbing folderInfo when running CheckDicomFolder", err)
			continue
		}
		logger.Println(folder, "is a valid dicom folder.")
		dicomFolders[folder] = folderInfo
	}
	for key, value := range dicomFolders {
		logger.Printf("folder info is %s\n%s\n", key, value)
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("Elapsed time: %.2f seconds for GetDicomFolders searching thru %s\n", elapsedTime.Seconds(), searchFolder)
	return dicomFolders, nil
}

// takes a parent dicomFolderPath that contains subfolders and returns a map with keys[manufacturerModelname,Scan type(PANO,CT,SCENE,etc..),FOV(if applicable)] and values [Folder path,fov size, name of scanner]
// example map[CT:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344460687/1.2.392.200036.9163.41.127414021.344460687.8332.1 FOV:15X15 ManufacturerModelName:[PreXion3D Explorer] PANO:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344460687/1.2.392.200036.9163.41.127414021.344460687.11336.1]
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
	logger.Printf("------- Start of CheckDicomFolder Script ---------")
	logger.Printf("checking to see if %s is a regular CT Scan or others", dicomFolderPath)
	//grab the subfolders in the parent folder to check which sub folder is which type of scan
	subFolderList, err := ListDirectories(dicomFolderPath)
	if err != nil {
		fmt.Println("Error getting subfolderList CheckDicomFolder:", err)
		logger.Printf("------- End of CheckDicomFolder Script ---------\n\n")
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
			logger.Printf("------- End of CheckDicomFolder Script ---------\n\n")
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
	if len(folderInfo) == 0 {
		log := "no scans were found in:%s"
		logger.Printf(log, dicomFolderPath)
		logger.Printf("------- End of CheckDicomFolder Script ---------\n\n")
		return nil, fmt.Errorf(log, dicomFolderPath)
	}
	logger.Printf("folderInfo is :%s\n\n\n", folderInfo)
	logger.Printf("------- End of CheckDicomFolder Script ---------\n\n")
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("Elapsed time: %.2f seconds for CheckDicomFolder on %s\n", elapsedTime.Seconds(), dicomFolderPath)
	return folderInfo, nil
}

// takes a dicomFile and checks to see what type of scan it is returns map with the scan details.
// example results -> map[ManufacturerModelName:[PreXion3D Explorer PRO] PANO:/Users/harrymbp/Developer/Projects/PreXion/temp/1.2.392.200036.9163.41.127414021.344261765/1.2.392.200036.9163.41.127414021.344261765.10632.1 PatientBirthDate:[20000101] PatientID:[07301985jc] PatientName:[Case^Number 6]]
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
	logger.Printf("------- Start of CheckScanType Script ---------")
	logger.Printf("checking to see what type of scan %s is", dicomFilePath)
	//check to see if file path given is a valid dicom
	dicomInfo, err := DicomInfoGrabber(dicomFilePath)
	if err != nil {
		logger.Printf("%s is not a valid dicom file", dicomFilePath)
		logger.Printf("------- End of CheckScanType Script ---------\n\n")
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
		//patient info we need to extract.
		PatientInfo := map[string]string{
			"PatientName":      "(0010,0010)",
			"PatientID":        "(0010,0020)",
			"PatientBirthDate": "(0010,0030)",
		}

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
				logger.Printf("SOPClassUID is: %s, Possibly Pano or Saved SCENE", value)
				ImageType := "(0008,0008)"
				//ImageType := "(0008,0008)"
				//referrence
				//"[ORIGINAL PRIMARY AXIAL]" - Regular CT Scan
				//"[ORIGINAL PRIMARY ]" - Pano or Ceph Scans.
				//"[DERIVED SECONDARY TERARECON]" - Saved SCENE.
				image, found := dicomInfo[ImageType]
				if found {
					switch image {
					case "[ORIGINAL PRIMARY ]": //pano
						logger.Printf("Scan is a %s %s", scanType, image)
						scanType = "PANO"
					case "[ORIGINAL SECONDARY SINGLEPLANE]": //pano from eclipse
						logger.Printf("Scan is a %s %s", scanType, image)
						scanType = "PANO"
					case "[DERIVED SECONDARY TERARECON]": //SCENE
						scanType = "SCENE"
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
		if foundModel && modelName != "[AQNET]" {
			logger.Printf("found Model name %s\n", modelName)
			dicomContents["ManufacturerModelName"] = modelName
		}
		for tag, element := range PatientInfo {
			value, found := dicomInfo[element]
			if found {
				dicomContents[tag] = value
			}
		}

	}
	dicomContents[scanType] = path
	//if there is a CT scan grab the FOV Value
	if scanType == "CT" {
		fovSize, err := GetFOVSize(dicomInfo, path)
		if err != nil {
			logger.Printf("%s is not a valid dicom file", dicomFilePath)
			return nil, err
		}
		dicomContents["FOV"] = fovSize
	}
	logger.Printf("end of script content is\n%s\n", dicomContents)
	logger.Printf("------- End of CheckScanType Script ---------\n\n")
	return dicomContents, nil
}

// takes dicomInfo and path string and calculates the FOV used and returns the string with the FOV size example "15X15"
func GetFOVSize(dicomInfo map[string]string, path string) (string, error) {
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
	logger.Printf("Starting function GetFOVSize on \n%s\n", path)
	//setting varible
	var fovString string
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
		fovString = xValue + "X" + yValue
		logger.Printf("Xvalue is currently:%s YValue is currently:%s", xValue, yValue)

	} else {
		logger.Printf("unable to find FOV!")
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	logger.Printf("Elapsed time: %.2f seconds for GetFOVSize\n", elapsedTime.Seconds())
	return fovString, nil
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

// takes a file and output folder path then copies the files over.
func copyFile(inputFile, outputPath string) error {
	// Open the source file
	sourceFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error opening source file:", err)
		return err
	}
	defer sourceFile.Close()
	fileName := filepath.Base(inputFile)
	//if file is .DS_Store ignore copying over the file.
	if fileName == ".DS_Store" {
		fmt.Printf("ignoring .DS_Store")
		return nil
	}
	outputFile := filepath.Join(outputPath, fileName)
	// Create or open the destination file
	destinationFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating destination file:", err)
		return err
	}
	defer destinationFile.Close()

	// Copy the content from the source to the destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		fmt.Println("Error copying file:", err)
		return err
	}

	// fmt.Println("File copied successfully.")
	return nil
}

// // takes in dicomFolderPath map[string]string ([dicomfolder]outputFolder), detailList | Open the file and modify it then save to output Path
// func SampleFunction(dicomFolder, outputFolder string) (string, error) {
// 	// Block of code for logger
// 	startTime := time.Now()
// 	// creates a logger for log files.
// 	logFileName := "logs/SampleFunction.txt"
// 	logger, logFile, err := createLogger(logFileName)
// 	if err != nil {
// 		fmt.Println("Error making log file for SampleFunction:", err)
// 		return "error", err
// 	}
// 	defer logFile.Close()
// 	// start of script
// 	logger.Printf("------- Start of SampleFunction Script ---------")
// 	// main function starts here
// 	// main function ends here
// 	endTime := time.Now()
// 	elapsedTime := endTime.Sub(startTime)
// 	logger.Printf("------- End of SampleFunction Script ---------\n\n")
// 	fmt.Printf("Elapsed time: %.2f seconds for SampleFunction\n", elapsedTime.Seconds())
// 	return "string", nil
// }
