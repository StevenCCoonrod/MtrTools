package main

import (
	"flag"
	"fmt"
	"mtrTools/dataObjects"
	"mtrTools/sqlDataAccessor"
	"mtrTools/sshDataAccess"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

var _SyncboxList []string

func main() {

	syncboxes, startTime, endTime, dcFilter, hostname := initialize()

	if len(os.Args) > 1 {

		switch {
		case IsFlagPassed("start") && IsFlagPassed("end"):
			mtrSweep(startTime, endTime, dcFilter, syncboxes)
		case IsFlagPassed("start") && !IsFlagPassed("end"):
			start := time.Since(time.Now().UTC().Add(-(startTime + (time.Minute * 3))))
			end := time.Since(time.Now().UTC().Add(-(startTime - (time.Minute * 3))))
			mtrSweep(start, end, dcFilter, syncboxes)
		default:
			if IsFlagPassed("a") {
				//No Time Frame Functions on ALL boxes
				fullMtrRetrievalCycle()
			} else {
				//No Time Frame Functions
				if IsFlagPassed("host") {
					getHostnameReport(hostname)
				}
				if len(syncboxes) > 0 {
					var reports []dataObjects.MtrReport
					for _, s := range syncboxes {
						reports = append(reports, getMtrData(
							s,
							time.Since(time.Now().AddDate(0, 0, -1)),
							time.Since(time.Now()),
							dcFilter)...)

						fmt.Println("Reports found:", len(reports))
					}

				}
			}
		}
	} else {
		//No args given
		//Use this to target a problem box or method
		programDisplay()
	}
}

// Retrieves data based on host name
func getHostnameReport(hostname string) {
	dataReturned := sqlDataAccessor.SelectMtrReports_ByHostname(hostname)
	var distinctBoxes []string
	var distinctDC []string
	var loss float32
	// Get distinct Syncboxes and target Data Centers, calculate average packet loss
	for _, r := range dataReturned {
		if !slices.Contains(distinctBoxes, r.SyncboxID) {
			distinctBoxes = append(distinctBoxes, r.SyncboxID)
		}
		if !slices.Contains(distinctDC, r.DataCenter) {
			distinctDC = append(distinctDC, r.DataCenter)
		}
		for _, h := range r.Hops {
			loss += h.PacketLoss
		}
	}
	fmt.Println("Reports with host hop:", len(dataReturned))
	averageLoss := loss / float32(len(dataReturned))
	fmt.Print("Destination Data Centers: ")
	for _, d := range distinctDC {
		fmt.Print(strings.ToUpper(d) + " ")
	}
	fmt.Println("\nAverage Loss:", averageLoss)
	fmt.Println("Syncboxes routing through host hop:")
	for _, s := range distinctBoxes {
		fmt.Println("\t", s)
	}
}

// Validates Timeframe and Initializes Mtr Retrieval
func mtrSweep(startTime time.Duration, endTime time.Duration, DCFilter string, syncboxes []string) {
	validTimes := validateTimeframe(startTime, endTime)
	if validTimes {
		var mtrReports []dataObjects.MtrReport
		if IsFlagPassed("a") {
			// Timeframe Functions on ALL boxes
			fmt.Println("Initiating Sweep at ", time.Now().UTC())
			for _, s := range _SyncboxList {
				mtrReports = getMtrData(s, startTime, endTime, DCFilter)
			}

		} else {
			// Timeframe Functions on Specific boxes
			for _, s := range syncboxes {
				mtrReports = getMtrData(s, startTime, endTime, DCFilter)
			}

		}

		if IsFlagPassed("p") { // Print to Console
			for _, r := range mtrReports {
				fmt.Println(r.PrintReport())
			}
		}

		if IsFlagPassed("pf") { // Print to File
			printReportsToTextFile(mtrReports)
		}
	}
}

// Check SSH, Insert new reports into DB, Select from DB.
// Retrieves all log files for the specified boxes within the timeframe provided
func getMtrData(syncbox string, startTime time.Duration, endTime time.Duration, DCFilter string) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batch []dataObjects.MtrReport
	var syncboxStatus string
	//Get datetimes based on provided durations
	start := time.Now().UTC().Add(-startTime)
	end := time.Now().UTC().Add(-endTime)

	//Print Console Header
	if !IsFlagPassed("a") {
		fmt.Println("Start Time:\t" + fmt.Sprint(start.Format(time.ANSIC)) +
			"\nEnd Time:\t" + fmt.Sprint(end.Format(time.ANSIC)))

	}
	if IsFlagPassed("dc") {
		fmt.Println("Data Center:\t" + strings.ToUpper(DCFilter))
	}

	//For each syncbox provided, Check SSH, Insert any new reports, and return all reports found in DB

	//Check SSH
	batch, syncboxStatus = sshDataAccess.GetMtrData_SpecificTimeframe(syncbox, start, end)
	//Insert any new reports into the DB
	insertMtrReportsIntoDB(batch)

	//Select the matching reports from the DB
	if IsFlagPassed("dc") {
		batch = sqlDataAccessor.SelectMtrReports_BySyncbox_DCAndTimeframe(syncbox, start, end, DCFilter)
		if len(batch) == 0 {
			fmt.Println("Reports found for "+syncbox+" going to "+strings.ToUpper(DCFilter)+":", len(batch), ". "+syncboxStatus)
		} else {
			fmt.Println("Reports found for "+syncbox+" going to "+strings.ToUpper(DCFilter)+":", len(batch))
		}

	} else {
		batch = sqlDataAccessor.SelectMtrReports_BySyncbox_Timeframe(syncbox, start, end)
		if len(batch) == 0 {
			fmt.Println("Reports found for "+syncbox+":", len(batch), " -- "+syncboxStatus)
		} else {
			fmt.Println("Reports found for "+syncbox+":", len(batch))
		}
	}
	mtrReports = append(mtrReports, batch...)

	return mtrReports
}

//^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^   Core Functions    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\\
//|||||||||||||||||||||||||||||||||||||=====================||||||||||||||||||||||||||||||||||||||||||
//vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv Secondary Functions vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\\

// Retrieves Syncbox list, establishes flag values and Syncbox args
func initialize() ([]string, time.Duration, time.Duration, string, string) {
	//Update the SyncboxList []string
	updateSyncboxList()

	startTime, endTime, dcFilter, hostname := setFlags()
	syncboxArgs := flag.Args()
	syncboxes := processSyncboxArgs(syncboxArgs)

	return syncboxes, startTime, endTime, dcFilter, hostname
}

// Takes a slice of MTR Reports, checks if each is already in the DB, if not it inserts it
func insertMtrReportsIntoDB(mtrReports []dataObjects.MtrReport) {
	if len(mtrReports) > 0 {
		//Check if the Report already exists in the DB
		successfulInsert := sqlDataAccessor.InsertMtrReports(mtrReports)

		if !successfulInsert {
			fmt.Println("Error inserting reports.")
		} else {
			fmt.Println(len(mtrReports), "inserted into the DB")
		}
	}
}

// Compares the DB and SSH Server list of Syncboxes
// and adds any that aren't in the DB to the DB.
// Updates the SyncboxList
func updateSyncboxList() {
	var sshSyncboxList []string
	//Get the list of Syncboxes currently in the DB
	dbSyncboxList := sqlDataAccessor.SelectAllSyncboxes()
	//Get list of Syncboxes currently on SSH server
	sshSyncboxList = sshDataAccess.GetSyncboxList()
	//If there's a difference in the number of Syncboxes in either list
	if len(sshSyncboxList) != 0 && len(dbSyncboxList) != len(sshSyncboxList) {
		//If the DB list doesn't contain the ssh syncbox, insert it into the DB
		for _, s := range sshSyncboxList {
			if !slices.Contains(dbSyncboxList, strings.ToUpper(s)) {
				sqlDataAccessor.InsertSyncbox(s)
			}
		}
		//Select the updated DB Syncbox list
		dbSyncboxList = sqlDataAccessor.SelectAllSyncboxes()
	}
	//Set the _SyncboxList equal to the DB list
	_SyncboxList = dbSyncboxList
}

// Prints all reports provided to a text file
func printReportsToTextFile(reports []dataObjects.MtrReport) {
	directory, errD := os.Getwd()
	newFilename := fmt.Sprint(directory + "\\MtrResults\\" + time.Now().Format("2006-01-02_03-04-PM") + "_mtrReport.txt")
	if errD != nil {
		fmt.Println("There was an error getting the working directory.\n", errD.Error())
	}
	file, err := os.Create(newFilename)
	if err != nil {
		fmt.Println("There was an error creating the text file.\n", err.Error())
	} else {
		defer file.Close()
		fmt.Println(newFilename)
		for _, r := range reports {
			var err2 error
			f, err2 := os.OpenFile(newFilename, os.O_APPEND|os.O_WRONLY, 0644)
			if err2 != nil {
				file.Close()
				fmt.Println(err2.Error())
			}

			_, err3 := fmt.Fprintln(f, r.PrintReport())
			if err3 != nil {
				fmt.Println("There was an error printing reports to the file.\n", err3.Error())
			}
		}
	}
}
