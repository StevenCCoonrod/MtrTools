package main

import (
	"fmt"
	"mtrTools/dataObjects"
	"mtrTools/sqlDataAccessor"
	"mtrTools/sshDataAccess"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slices"
)

//This .go file holds the majority of code involved in the primary Mtr Data Collection Process.
//It retrieves all MTR data from the specified Server via SSH,
//Iterating through the directories of every syncbox for the most recently added files,
//Parsing the data collected, and inserting it into the target Database

var _baseDirectory string = "/var/log/syncbak/catcher-mtrs/"

// Retrieves ALL MTR logs for ALL syncboxes in the SyncboxList
func fullMtrRetrievalCycle() {
	for {
		updateSyncboxList()
		timeOfInitiation := time.Now()
		fmt.Println("============ Initiating Full MTR Sweep ============")
		fmt.Println("\nFull Sweep Initiated At", timeOfInitiation.Format(time.ANSIC))

		//Divide the list of Syncboxes to be iterated through into batches (a slice of string slices)
		batchCount := 15
		batches := make([][]string, len(_SyncboxList)/batchCount+1)
		position := 0
		for i, s := range _SyncboxList {
			if i != 0 && i%batchCount == 0 {
				position += 1
			}
			batches[position] = append(batches[position], s)
		}

		//Create a Waitgroup to sync channels.
		//Waitgroup is added in each channel and set to Done in the Go Routine
		var wg sync.WaitGroup
		ch := make(chan []string)
		//Make a channel for each batch, to send each batch through
		go func() {
			for _, b := range batches {
				ch <- b
			}
			close(ch)
		}()

		//For each batch of syncboxes ran through the channel...
		var batchNumber int
		for batch := range ch {
			wg.Add(1)
			batchNumber += 1
			fmt.Println("--> Working on batch "+fmt.Sprint(batchNumber)+":", batch[0], "-", batch[len(batch)-1])
			go getBatchMtrData(&wg, batch, batchNumber, timeOfInitiation.Add(time.Minute*-30), timeOfInitiation)
			//Sleep timer needed to space out connections and avoid errors
			time.Sleep(time.Second * 2)
		}

		//Wait for all batches to be collected
		wg.Wait()
		fmt.Println("============ MTR Sweep Completed ============")
		fmt.Println("Cycle Duration:", time.Since(timeOfInitiation))
	}

}

// Updates the Syncboxes in the DB if any new ones are found in the SSH directory
func updateSyncboxList() {
	fmt.Println("Updating syncbox list...")
	var updatedList []string
	dbSyncboxList := sqlDataAccessor.SelectAllSyncboxes()
	syncboxList := getSyncboxList()

	for _, s := range syncboxList {
		if !slices.Contains(dbSyncboxList, strings.ToUpper(s)) {
			sqlDataAccessor.InsertSyncbox(s)
		}
	}
	dbSyncboxList = sqlDataAccessor.SelectAllSyncboxes()
	for i, s := range dbSyncboxList {
		if !slices.Contains(syncboxList, strings.ToLower(s)) {
			updatedList = removeSliceElement(dbSyncboxList, i)
		}
		// if strings.Contains(s, "-2309") {
		// 	updatedList = append(updatedList, s)
		// }
	}
	if updatedList != nil {
		_SyncboxList = updatedList
	}

}

// Currently being ran as a go routine inside a channel.
// Takes the reference to the process' waitgroup, the batch of syncboxes, the number of the batch for tracking,
// and the start and end times for the sweep
func getBatchMtrData(wg *sync.WaitGroup, syncboxes []string, batchNumber int, start time.Time, end time.Time) []dataObjects.MtrReport {

	var batchReports []dataObjects.MtrReport

	//Because of the directory structure on the server, a different command needs to be ran for each day involved
	//This loop ensures that the command for every day involved in the search is parsed properly
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		batchReports = append(batchReports, GetBatchSyncboxMtrReports(syncboxes, d)...)
	}
	if batchReports == nil {
		fmt.Println("***No reports returned")
	}

	//Insert reports into the DB
	fmt.Println("... Inserting into DB for batch "+fmt.Sprint(batchNumber)+":", syncboxes[0], "-", syncboxes[len(syncboxes)-1])

	reportsInserted := insertMtrReportsIntoDB(batchReports)
	wg.Done()
	fmt.Println("||| BATCH "+
		fmt.Sprint(batchNumber)+":", syncboxes[0], "-", syncboxes[len(syncboxes)-1]+
		" COMPLETED\t", reportsInserted, " new reports inserted.")
	return batchReports
}

// Main assembler of the data collection. Establishes an SSH client and initiates the 3 step process
//  1. Get the filenames of all the most recent logs in each Syncbox directory
//  2. Retrieve all the data for every log file found in each directory
//  3. Parse the data into MtrReport structs to be inserted into the DB
func GetBatchSyncboxMtrReports(syncboxes []string, targetDate time.Time) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batchReports []dataObjects.MtrReport
	// conn := sshDataAccess.ConnectToSSH()

	// if conn != nil {
	// defer conn.Close()
	mtrLogFilenames := getBatchSyncboxLogFilenamesLocal(syncboxes, targetDate)

	if len(mtrLogFilenames) > 0 {
		rawMtrData, targetDCs := getBatchSyncboxLocalMtrData(syncboxes, mtrLogFilenames, targetDate)
		// fmt.Println("Got Log Data...", len(rawMtrData))
		mtrReports = parseRawDataIntoMtrReport(rawMtrData, targetDCs)
		// fmt.Println("Parsed data into reports...", len(mtrReports))

		batchReports = append(batchReports, mtrReports...)
	}
	// }

	return batchReports
}

type CommandRequest struct {
	Command string
	Args    string
}

// Step 1 in the MTR Data collection process.
// Retrieves the most recently added log file names found in each syncbox directory.
func getBatchSyncboxLogFilenamesLocal(syncboxes []string, targetDate time.Time) []string {
	validMonth := sshDataAccess.ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := sshDataAccess.ValidateDateField(fmt.Sprint(targetDate.Day()))
	// var command string
	var dataReturned string
	filesToRetrieve := 20
	for _, s := range syncboxes {
		var commandRequest CommandRequest
		commandRequest.Command = "cd"
		commandRequest.Args = _baseDirectory +
			fmt.Sprint(targetDate.Year()) + "/" +
			validMonth + "/" +
			validDay + "/" + strings.ToLower(s) +
			" && ls -t | head -" + fmt.Sprint(filesToRetrieve)

		// command = "cd " + _baseDirectory +
		// 	fmt.Sprint(targetDate.Year()) + "/" +
		// 	validMonth + "/" +
		// 	validDay + "/" + strings.ToLower(s) +
		// 	" && ls -t | head -" + fmt.Sprint(filesToRetrieve)
		dataReturned_1, err := runLocalCommand(commandRequest)
		if err != nil {
			if strings.Contains(err.Error(), "Process exited with status 1") {
				//No log files in directory. No issue.
			} else {
				fmt.Println(err.Error())
			}
		}
		if len(dataReturned_1) > 0 {
			dataReturned += dataReturned_1
		}
	}

	return strings.Split(dataReturned, "\n")
}

func getBatchSyncboxLocalMtrData(syncboxes []string, mtrLogFilenames []string, targetDate time.Time) ([]string, []string) {
	validMonth := sshDataAccess.ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := sshDataAccess.ValidateDateField(fmt.Sprint(targetDate.Day()))
	// var batchDataString string

	var rawReports []string
	// Target each Syncbox directory in this batch, build and run a command for each log file provided
	for _, s := range syncboxes {

		// var command string
		var dataReturned string
		var commandRequest CommandRequest

		for _, l := range mtrLogFilenames {
			var err error
			// Check that the filename contains the syncbox name so that only the data of log files for this box is returned
			if strings.Contains(l, strings.ToLower(s)) {
				// Build a command targeting this specific log file in the target Syncboxes directory
				commandRequest.Command = "cat"
				commandRequest.Args = sshDataAccess.BaseDirectory +
					fmt.Sprint(targetDate.Year()) + "/" +
					validMonth + "/" +
					validDay + "/" + strings.ToLower(s) + "/" + l
				// command += l
				// command = "cat " + sshDataAccess.BaseDirectory +
				// 	fmt.Sprint(targetDate.Year()) + "/" +
				// 	validMonth + "/" +
				// 	validDay + "/" + strings.ToLower(s) + "/"
				// command += l

				// Run the command
				dataReturned, err = runBatchMtrLocalCommand(commandRequest)
				if err != nil {
					if strings.Contains(err.Error(), "Process exited with status 1") {
						//Do nothing. No data returned.
					} else {
						fmt.Println("Error running command on SSH Server.\n" + err.Error())
					}
				} else {
					// Append the log data to the batch data string
					rawReports = append(rawReports, dataReturned)
				}
			}
		}
	}

	return rawReports, mtrLogFilenames
}

func testFunction() {
	commandRequest := CommandRequest{Command: "sh", Args: "cd C:/Users/WakingBear/Desktop/NetOps/Go/NetopsMtrTools/ | dir"}
	dataReturned, err := runBatchMtrLocalCommand(commandRequest)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(dataReturned)
	}
}

// Uses an ssh connection and runs the given command, returning any data and errors
func runBatchMtrLocalCommand(commandRequest CommandRequest) (string, error) {

	dataReturned, err := runLocalCommand(commandRequest)
	if err != nil {
		fmt.Println(err.Error())
	}

	return dataReturned, err
}

func runLocalCommand(commandRequest CommandRequest) (string, error) {

	// cmd := exec.Command(commandRequest.Command, commandRequest.Args)
	// var dataReturned []byte
	// var err error
	// if dataReturned, err = cmd.Output(); err != nil {
	// 	fmt.Println(err.Error())
	// }
	dataReturned, err := exec.Command(commandRequest.Command, commandRequest.Args).Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(dataReturned)
	return string(dataReturned), err
}

// Takes a slice of MTR Reports, checks if each is already in the DB, if not it inserts it.
// Returns the count of new reports inserted into the DB
func insertMtrReportsIntoDB(mtrReports []dataObjects.MtrReport) int {
	var reportsInsertedIntoDB int
	if len(mtrReports) > 0 {
		//Stored procedure makes the check for if the report already exists
		reportsInsertedIntoDB = sqlDataAccessor.InsertMtrReports(mtrReports)

		// fmt.Println(reportsInsertedIntoDB, "reports inserted into the DB")

	}
	return reportsInsertedIntoDB
}

func getSyncboxList() []string {

	date := time.Now()
	validMonth := validateDateField(fmt.Sprint(int32(date.Month())))
	validDay := validateDateField(fmt.Sprint(date.Day()))
	var syncboxList []string
	var commandRequest CommandRequest
	commandRequest.Command = "ls"
	commandRequest.Args = _baseDirectory +
		fmt.Sprint(date.Year()) + "/" +
		validMonth + "/" + validDay + "/"

	// command := "ls " + _baseDirectory +
	// 	fmt.Sprint(date.Year()) + "/" +
	// 	validMonth + "/" + validDay + "/"

	dataReturned, err := runLocalCommand(commandRequest)
	if err != nil {
		fmt.Println(err)
	}
	//defer conn.Close()

	tempSyncboxList := strings.Split(dataReturned, "\n")
	// for _, s := range tempSyncboxList {
	// 	if strings.Contains(s, "-2309") {
	// 		syncboxList = append(syncboxList, s)
	// 	}
	// }
	for _, s := range tempSyncboxList {
		if len(strings.TrimSpace(s)) > 0 {
			syncboxList = append(syncboxList, s)
		}
	}

	// return tempSyncboxList
	return syncboxList
}

// Parses raw MTR data into a slice of MtrReports
func parseRawDataIntoMtrReport(rawData []string, LogFilenames []string) []dataObjects.MtrReport {

	//Create the Report array to hold all the retrieved mtr Reports
	var mtrReports []dataObjects.MtrReport

	var currentReportsTargetDC string
	var currentReportsStartTime time.Time
	var currentReportsHost string

	// Loop through each raw report string and parse into an MtrReport object
	// m = Single Mtr Raw Data
	for h, m := range rawData {
		currentDataLogFilename := LogFilenames[h]
		logfilenameFields := strings.Split(currentDataLogFilename, "-")
		if strings.Contains(currentDataLogFilename, "-2309") {
			currentReportsHost = logfilenameFields[0] + "-" + logfilenameFields[1]
			currentReportsStartTime = time.Date(parseStringToInt(logfilenameFields[2]),
				time.Month(parseStringToInt(logfilenameFields[3])),
				parseStringToInt(logfilenameFields[4]),
				parseStringToInt(logfilenameFields[5]),
				parseStringToInt(logfilenameFields[6]), 0, 0, time.UTC)
			currentReportsTargetDC = logfilenameFields[7]

		} else {
			currentReportsHost = logfilenameFields[0]
			currentReportsStartTime = time.Date(parseStringToInt(logfilenameFields[1]),
				time.Month(parseStringToInt(logfilenameFields[2])),
				parseStringToInt(logfilenameFields[3]),
				parseStringToInt(logfilenameFields[4]),
				parseStringToInt(logfilenameFields[5]), 0, 0, time.UTC)
			currentReportsTargetDC = logfilenameFields[6]
		}

		if m != "" && !strings.Contains(m, "<") {
			//Create new mtrReport
			mtrReport := dataObjects.MtrReport{}
			mtrReport.SyncboxID = currentReportsHost
			mtrReport.DataCenter = currentReportsTargetDC
			mtrReport.StartTime = currentReportsStartTime
			//Split data into lines
			lines := strings.Split(m, "\n")
			//Iterate through each line in the data
			for _, l := range lines {

				//If its the first line, parse the StartTime datetime
				if !strings.Contains(strings.ToLower(l), "start") && !strings.Contains(strings.ToLower(l), "host") {
					mtrReport = parseHopsForReport(l, mtrReport)
				}
			}

			var lastHopDataCenter string
			if len(mtrReport.Hops) > 0 {
				//Verify if traceroute was successful
				lastHopHost := mtrReport.Hops[len(mtrReport.Hops)-1].Hostname
				if strings.Contains(lastHopHost, "util") {
					lastHopDataCenter = strings.Replace(lastHopHost, "util", "", 1)
					lastHopDataCenter = strings.Replace(lastHopDataCenter, "eqnx", "", 1)
				}
			}
			if strings.EqualFold(lastHopDataCenter, mtrReport.DataCenter) {
				mtrReport.Success = true
			} else {
				mtrReport.Success = false
			}

			mtrReports = append(mtrReports, mtrReport)
		}
	}

	return mtrReports
}

func parseHopsForReport(l string, mtrReport dataObjects.MtrReport) dataObjects.MtrReport {

	//Create new hop
	hop := dataObjects.MtrHop{}
	//Split the line by fields and parse a new hop
	f := strings.Fields(l)

	//Painful way of checking that fields are not null
	if len(f) > 0 {
		var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-z0-9 ]+`)
		hn := nonAlphanumericRegex.ReplaceAllString(f[0], "")

		hop.HopNumber = parseStringToInt(hn)
		if len(f) > 1 {
			hop.Hostname = f[1]
		}
		if len(f) > 2 {
			pl := strings.Replace(f[2], "%", "", 1)
			hop.PacketLoss = parseStringToFloat32(pl)
		}
		if len(f) > 3 {
			hop.PacketsSent = parseStringToInt(f[3])
		}
		if len(f) > 4 {
			hop.LastPing = parseStringToFloat32(f[4])
		}
		if len(f) > 5 {
			hop.AveragePing = parseStringToFloat32(f[5])
		}
		if len(f) > 6 {
			hop.BestPing = parseStringToFloat32(f[6])
		}
		if len(f) > 7 {
			hop.WorstPing = parseStringToFloat32(f[7])
		}
		if len(f) > 8 {
			hop.StdDev = parseStringToFloat32(f[8])
		}
		mtrReport.Hops = append(mtrReport.Hops, hop)
	}
	return mtrReport
}

// Helper method to parse strings into a float32
func parseStringToFloat32(s string) float32 {
	var pl float64
	var err error
	if s != "" {
		pl, err = strconv.ParseFloat(s, 32)
		if err != nil {
			fmt.Println(err.Error())
		}
	} else {
		pl = 0.0
	}

	return float32(pl)
}

// Helper method to parse strings into an int
func parseStringToInt(s string) int {
	var i int
	var err error

	if s != "" {
		i, err = strconv.Atoi(s)
		if err != nil {
			fmt.Println(err.Error())
		}
	} else {
		i = 0
	}

	return i
}
