package main

import (
	"bytes"
	"fmt"
	"mtrTools/dataObjects"
	"mtrTools/sqlDataAccessor"
	"mtrTools/sshDataAccess"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"
)

//This .go file holds the majority of code involved in the primary Mtr Data Collection Process.
//It retrieves all MTR data from the specified Server via SSH,
//Iterating through the directories of every syncbox for the past 24 hours,
//Parsing the data collected, and inserting it into the target Database

// Retrieves ALL MTR logs for ALL syncboxes in the SyncboxList
func fullMtrRetrievalCycle() {

	timeOfInitiation := time.Now().UTC()
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
		fmt.Println("Working on batch "+fmt.Sprint(batchNumber)+":", batch[0], "-", batch[len(batch)-1])
		go getBatchMtrData(&wg, batch, batchNumber, timeOfInitiation.AddDate(0, 0, -1), timeOfInitiation)
		//Sleep timer needed to space out connections and avoid errors
		time.Sleep(time.Second * 10)
	}

	//Wait for all batches to be collected
	wg.Wait()
	fmt.Println("============ MTR Sweep Completed ============")
	fmt.Println("Cycle Duration:", time.Since(timeOfInitiation))
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
	fmt.Println("Inserting into DB for batch "+fmt.Sprint(batchNumber)+":", syncboxes[0], "-", syncboxes[len(syncboxes)-1])
	insertMtrReportsIntoDB(batchReports)
	wg.Done()
	fmt.Println("BATCH "+fmt.Sprint(batchNumber)+":", syncboxes[0], "-", syncboxes[len(syncboxes)-1]+" COMPLETED")
	return batchReports
}

// Main assembler of the data collection. Establishes an SSH client and initiates the 4 step process
//  1. Get the filenames of all the log files in each directory
//  2. Pull all the data for every log file found in each directory
//  3. Parse the data into MtrReport structs
//  4. Match the parsed data with its corresponding filename and set it as the report's ID
func GetBatchSyncboxMtrReports(syncboxes []string, targetDate time.Time) []dataObjects.MtrReport {
	var validatedMtrReports []dataObjects.MtrReport
	var batchReports []dataObjects.MtrReport
	conn := sshDataAccess.ConnectToSSH()

	if conn != nil {
		defer conn.Close()
		mtrLogFilenames, err := getBatchSyncboxLogFilenames(conn, syncboxes, targetDate)
		if err != nil {
			//Do Nothing. This means one or more of the Syncbox directories returned no data
			//Other directories still may have returned data
		}
		if len(mtrLogFilenames) > 0 {
			rawMtrData := getBatchSyncboxMtrData(conn, syncboxes, targetDate)
			fmt.Println("Got Log Data...", len(rawMtrData))
			tempMtrReports := sshDataAccess.ParseSshDataIntoMtrReport(rawMtrData)
			fmt.Println("Parsed data into reports...", len(tempMtrReports))
			validatedMtrReports = matchBatchMtrDataWithFilenames(mtrLogFilenames, tempMtrReports)
			fmt.Println("Validated Report ID's... ", len(validatedMtrReports))

			batchReports = append(batchReports, validatedMtrReports...)
		}
	}

	return batchReports
}

// Step 1 in the MTR Data collection process. Retrieves the log file names found in each syncbox directory.
func getBatchSyncboxLogFilenames(conn *ssh.Client, syncboxes []string, targetDate time.Time) ([]string, error) {
	validMonth := sshDataAccess.ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := sshDataAccess.ValidateDateField(fmt.Sprint(targetDate.Day()))

	command := "cd " + sshDataAccess.BaseDirectory +
		fmt.Sprint(targetDate.Year()) + "/" +
		validMonth + "/" +
		validDay + " && ls "

	for _, s := range syncboxes {
		command +=
			strings.ToLower(s) + " "
	}

	dataReturned_1, err := sshDataAccess.RunClientCommand(conn, command)

	return strings.Split(dataReturned_1, "\n"), err
}

// Step 2 in the MTR Data collection process. Retrieves all the data in each log file in each syncbox directory.
func getBatchSyncboxMtrData(conn *ssh.Client, syncboxes []string, targetDate time.Time) string {
	validMonth := sshDataAccess.ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := sshDataAccess.ValidateDateField(fmt.Sprint(targetDate.Day()))

	command := "cd " + sshDataAccess.BaseDirectory +
		fmt.Sprint(targetDate.Year()) + "/" +
		validMonth + "/" +
		validDay + " && cat "

	for _, s := range syncboxes {
		command += strings.ToLower(s) + "/" + "*.log "
	}
	dataReturned, err := runBatchMtrClientCommand(conn, command)
	if err != nil {
		if strings.Contains(err.Error(), "Process exited with status 1") {
			//Do nothing. This just means one of the Syncbox directories did not return any data
			//The other Syncbox directories in the batch may have returned data
		} else {
			fmt.Println("Error running command on SSH Server.\n" + err.Error())
		}
	}
	return dataReturned
}

// Step 4 in the MTR Data collection process. Matches the parsed MTR data retrieved with its corresponding filename.
func matchBatchMtrDataWithFilenames(mtrLogFilenames []string, tempMtrReports []dataObjects.MtrReport) []dataObjects.MtrReport {
	var validatedMtrReports []dataObjects.MtrReport
	for i, f := range mtrLogFilenames {
		if strings.Contains(f, "/var") || !strings.Contains(f, ".log") {
			slices.Delete(mtrLogFilenames, i, i+1)
		}
	}

	for _, l := range mtrLogFilenames {
		if l != "" && strings.Contains(l, ".log") {

			//Split each line on the "-" and parse the fields
			f := strings.Split(l, "-")
			LogFileBoxName := f[0] + "-" + f[1]
			dateYear := sshDataAccess.ParseStringToInt(f[2])
			dateMonth := time.Month(sshDataAccess.ParseStringToInt(f[3]))
			dateDay := sshDataAccess.ParseStringToInt(f[4])
			dateHour := sshDataAccess.ParseStringToInt(f[5])
			dateMinute := sshDataAccess.ParseStringToInt(f[6])
			dataCenter := f[7]

			//Parse this into a time.Time object
			logFileDateTime := time.Date(dateYear, dateMonth, dateDay, dateHour, dateMinute, 0, 0, &time.Location{})

			//Match the parsed datetime from the filename
			//with the corresponding report in the mtr list
			for _, r := range tempMtrReports {
				id := strings.ReplaceAll(l, " ", "-")
				// fmt.Println("SyncboxID: " + r.SyncboxID + "\tLog File Field" + LogFileBoxName)
				if r.SyncboxID == LogFileBoxName &&
					r.StartTime.Year() == logFileDateTime.Year() &&
					r.StartTime.Month() == logFileDateTime.Month() &&
					r.StartTime.Day() == logFileDateTime.Day() &&
					r.StartTime.Hour() == logFileDateTime.Hour() &&
					r.StartTime.Minute() == logFileDateTime.Minute() &&
					r.DataCenter == dataCenter {
					r.ReportID = id
					validatedMtrReports = append(validatedMtrReports, r)
					break
				}

			}
		}

	}
	return validatedMtrReports
}

// This takes an ssh connection and runs the given command, returning any data and errors
func runBatchMtrClientCommand(conn *ssh.Client, command string) (string, error) {
	var buff bytes.Buffer
	var err2 error
	if conn != nil {
		session, err := conn.NewSession()
		if err != nil {
			fmt.Println("Error beginning session on SSH Server.\n" + err.Error())
		}
		defer session.Close()

		session.Stdout = &buff
		err2 = session.Run(command)
	}

	return buff.String(), err2
}

// Takes a slice of MTR Reports, checks if each is already in the DB, if not it inserts it
func insertMtrReportsIntoDB(mtrReports []dataObjects.MtrReport) int {
	var reportsInsertedIntoDB int
	if len(mtrReports) > 0 {
		//Check if the Report already exists in the DB
		reportsInsertedIntoDB = sqlDataAccessor.InsertMtrReports(mtrReports)

		if reportsInsertedIntoDB > 0 == false {
			fmt.Println("Error inserting reports.")
		} else {
			fmt.Println(reportsInsertedIntoDB, "reports inserted into the DB")
		}
	}
	return reportsInsertedIntoDB
}
