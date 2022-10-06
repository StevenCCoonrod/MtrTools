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
//Iterating through the directories of every syncbox for the most recently added files,
//Parsing the data collected, and inserting it into the target Database

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
	var sshSyncboxList []string
	dbSyncboxList := sqlDataAccessor.SelectAllSyncboxes()
	sshSyncboxList = sshDataAccess.GetSyncboxList()
	if len(sshSyncboxList) != 0 && len(dbSyncboxList) != len(sshSyncboxList) {
		//If the DB list doesn't contain the ssh syncbox, insert it into the DB
		for _, s := range sshSyncboxList {
			if !slices.Contains(dbSyncboxList, strings.ToUpper(s)) {
				sqlDataAccessor.InsertSyncbox(s)
			}
		}
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
	conn := sshDataAccess.ConnectToSSH()

	if conn != nil {
		defer conn.Close()
		mtrLogFilenames := getBatchSyncboxLogFilenames(conn, syncboxes, targetDate)

		if len(mtrLogFilenames) > 0 {
			rawMtrData := getBatchSyncboxMtrData(conn, syncboxes, mtrLogFilenames, targetDate)
			// fmt.Println("Got Log Data...", len(rawMtrData))
			mtrReports = sshDataAccess.ParseSshDataIntoMtrReport(rawMtrData)
			// fmt.Println("Parsed data into reports...", len(mtrReports))

			batchReports = append(batchReports, mtrReports...)
		}
	}

	return batchReports
}

// Step 1 in the MTR Data collection process.
// Retrieves the most recently added log file names found in each syncbox directory.
func getBatchSyncboxLogFilenames(conn *ssh.Client, syncboxes []string, targetDate time.Time) []string {
	validMonth := sshDataAccess.ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := sshDataAccess.ValidateDateField(fmt.Sprint(targetDate.Day()))
	var command string
	var dataReturned string
	filesToRetrieve := 20
	for _, s := range syncboxes {
		command = "cd " + sshDataAccess.BaseDirectory +
			fmt.Sprint(targetDate.Year()) + "/" +
			validMonth + "/" +
			validDay + "/" + strings.ToLower(s) +
			" && ls -t | head -" + fmt.Sprint(filesToRetrieve)
		dataReturned_1, err := sshDataAccess.RunClientCommand(conn, command)
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

// Step 2 in the MTR Data collection process.
// Returns the log data for each log file found in the func getBatchSyncboxLogFilenames()
func getBatchSyncboxMtrData(conn *ssh.Client, syncboxes []string, mtrLogFilenames []string, targetDate time.Time) string {
	validMonth := sshDataAccess.ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := sshDataAccess.ValidateDateField(fmt.Sprint(targetDate.Day()))
	var batchDataString string

	// Target each Syncbox directory in this batch, build and run a command for each log file provided
	for _, s := range syncboxes {
		var command string
		var dataReturned string
		for _, l := range mtrLogFilenames {
			var err error
			// Check that the filename contains the syncbox name so that only the data of log files for this box is returned
			if strings.Contains(l, strings.ToLower(s)) {
				// Build a command targeting this specific log file in the target Syncboxes directory
				command = "cat " + sshDataAccess.BaseDirectory +
					fmt.Sprint(targetDate.Year()) + "/" +
					validMonth + "/" +
					validDay + "/" + strings.ToLower(s) + "/"
				command += l
				// Run the command
				dataReturned, err = runBatchMtrClientCommand(conn, command)
				if err != nil {
					if strings.Contains(err.Error(), "Process exited with status 1") {
						//Do nothing. This just means one of the Syncbox directories did not return any data
						//The other Syncbox directories in the batch may have returned data
					} else {
						fmt.Println("Error running command on SSH Server.\n" + err.Error())
					}
				} else {
					// Append the log data to the batch data string
					batchDataString += dataReturned
				}
			}
		}
	}

	return batchDataString
}

// Uses an ssh connection and runs the given command, returning any data and errors
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
