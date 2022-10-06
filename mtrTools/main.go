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
)

var _SyncboxList []string
var _SyncboxArgs []string
var _StartTime time.Duration
var _EndTime time.Duration
var _DataCenter string
var _Hostname string

func main() {

	initialize()

	if len(os.Args) > 1 {

		switch { // Timeframe MTR Searches
		case IsFlagPassed("start") && IsFlagPassed("end"):
			mtrSweep(_StartTime, _EndTime)
		case IsFlagPassed("start") && !IsFlagPassed("end"):
			start := time.Since(time.Now().UTC().Add(-(_StartTime + (time.Minute * 3))))
			end := time.Since(time.Now().UTC().Add(-(_StartTime - (time.Minute * 3))))
			mtrSweep(start, end)
		default: // Other Functions
			if IsFlagPassed("a") {
				fullMtrRetrievalCycle() //CORE MTR COLLECTION PROCESS
			} else {
				if IsFlagPassed("host") {
					getHostnameReport(_Hostname)
				}
				if len(_SyncboxArgs) > 0 {
					var reports []dataObjects.MtrReport
					reports = append(reports, mtrSweep(
						time.Since(time.Now().AddDate(0, 0, -1)),
						time.Since(time.Now()))...)

					fmt.Println("Reports found:", len(reports))
				}

			}
		}
	} else {
		//No Flags/Args provided
		updateSyncboxList()
		programDisplay()
	}
}

// Validates Timeframe and Initializes Mtr Retrieval
func mtrSweep(startTime time.Duration, endTime time.Duration) []dataObjects.MtrReport {

	searchStartTime := time.Now().UTC()
	start := searchStartTime.Add(-startTime)
	end := searchStartTime.Add(-endTime)
	validTimes := validateTimeframe(startTime, endTime)
	var mtrReports []dataObjects.MtrReport

	if validTimes {
		if IsFlagPassed("a") {
			// Timeframe Functions on ALL boxes
			display_TimeframeSearch_Header(searchStartTime, start, end, _DataCenter)
			for _, s := range _SyncboxList {
				mtrReports = append(mtrReports, getMtrData(s, searchStartTime, startTime, endTime, _DataCenter)...)
			}
		} else {
			display_TimeframeSearch_Header(searchStartTime, start, end, _DataCenter)
			// Timeframe Functions on Specific boxes
			for _, s := range _SyncboxArgs {
				mtrReports = append(mtrReports, getMtrData(s, searchStartTime, startTime, endTime, _DataCenter)...)
			}

		}

		if IsFlagPassed("p") { // Print to Console
			for _, r := range mtrReports {
				fmt.Println(len(r.Hops))
				fmt.Println(r.PrintReport())
			}
		}

		if IsFlagPassed("pf") { // Print to File
			printReportsToTextFile(mtrReports)
		}

	}

	searchReport(mtrReports)

	return mtrReports
}

// Check SSH, Insert new reports into DB, Select from DB.
// Retrieves all log files for the specified boxes within the timeframe provided
func getMtrData(syncbox string, searchStartTime time.Time, startTime time.Duration, endTime time.Duration, DCFilter string) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batch []dataObjects.MtrReport
	var syncboxStatus string
	//Get datetimes based on provided durations
	start := searchStartTime.Add(-startTime)
	end := searchStartTime.Add(-endTime)

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
func initialize() {
	_SyncboxList = sqlDataAccessor.SelectAllSyncboxes()
	_StartTime, _EndTime, _DataCenter, _Hostname = setFlags()

	_SyncboxArgs = processSyncboxArgs(flag.Args())
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
