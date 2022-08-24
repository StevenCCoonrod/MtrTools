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
		case isFlagPassed("start") && isFlagPassed("end"):
			timeframeFunctions(startTime, endTime, dcFilter, syncboxes)
		case isFlagPassed("start") && !isFlagPassed("end"):
			targetTimeFunctions(startTime, dcFilter, syncboxes)
		default:
			if isFlagPassed("a") {
				//No Time Frame Functions on ALL boxes
				fullMtrRetrievalCycle(dcFilter)
			} else {
				//No Time Frame Functions
				if isFlagPassed("host") {
					getHostnameReport(hostname)
				}
				if len(syncboxes) > 0 {
					reports := getMtrData_Timeframe(
						syncboxes,
						time.Since(time.Now().AddDate(0, 0, -1)),
						time.Since(time.Now()),
						dcFilter)
					fmt.Println("Reports found:", len(reports))
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
func timeframeFunctions(startTime time.Duration, endTime time.Duration, DCFilter string, syncboxes []string) {
	validTimes := validateTimeframe(startTime, endTime)
	if validTimes {
		var mtrReports []dataObjects.MtrReport
		if isFlagPassed("a") {
			// Timeframe Functions on ALL boxes
			fmt.Println("Initiating Full Timeframe Sweep at ", time.Now().UTC())
			mtrReports = getMtrData_Timeframe(_SyncboxList, startTime, endTime, DCFilter)
		} else {
			// Timeframe Functions on Specific boxes
			mtrReports = getMtrData_Timeframe(syncboxes, startTime, endTime, DCFilter)
		}

		if isFlagPassed("p") { // Print to Console
			for _, r := range mtrReports {
				fmt.Println(r.PrintReport())
			}
		}

		if isFlagPassed("pf") { // Print to File
			printReportsToTextFile(mtrReports)
		}
	}
}

// Validates Target Time and Initializes Mtr Retrieval
func targetTimeFunctions(startTime time.Duration, DCFilter string, syncboxes []string) {
	//Verify if start time is at least 5 minutes in the past
	if time.Now().Add(-startTime).Before(time.Now().Add(-time.Minute * 4)) {
		var mtrReports []dataObjects.MtrReport

		if isFlagPassed("a") { //Retrieve Mtr's for ALL Syncboxes
			mtrReports = getMtrData_TargetTime(_SyncboxList, startTime, DCFilter)
		} else { //Retrieve Mtr's for Specific Syncboxes
			mtrReports = getMtrData_TargetTime(syncboxes, startTime, DCFilter)
		}

		if isFlagPassed("p") { // Print to Console
			for _, r := range mtrReports {
				fmt.Println(r.PrintReport())
			}
		}

		if isFlagPassed("pf") { // Print to File
			printReportsToTextFile(mtrReports)
		}
	} else {
		fmt.Println("Start time must be at least 5 minutes ago.")
	}
}

// Retrieves ALL MTR logs for ALL syncboxes in the SyncboxList
func fullMtrRetrievalCycle(DCFilter string) {
	fmt.Println("============ Initiating Full MTR Sweep ============")
	fmt.Println("/nFull Sweep Initiated At ", time.Now().UTC())
	for _, s := range _SyncboxList {
		var currentSyncbox []string
		currentSyncbox = append(currentSyncbox, s)
		getMtrData_Timeframe(
			currentSyncbox, time.Since(time.Now().UTC().AddDate(0, 0, -1)), time.Duration(0), DCFilter)
	}

	fmt.Println("============ MTR Sweep Completed ============")
}

// Check SSH, Insert new reports into DB, Select from DB.
// Retrieves all log files for the specified boxes within the timeframe provided
func getMtrData_Timeframe(syncboxes []string, startTime time.Duration, endTime time.Duration, DCFilter string) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batch []dataObjects.MtrReport

	//Get datetimes based on provided durations
	start := time.Now().UTC().Add(-startTime)
	end := time.Now().UTC().Add(-endTime)

	//Print Console Header
	fmt.Println("Start Time:\t" + fmt.Sprint(start.Format(time.ANSIC)) +
		"\nEnd Time:\t" + fmt.Sprint(end.Format(time.ANSIC)))
	if isFlagPassed("dc") {
		fmt.Println("Data Center:\t" + strings.ToUpper(DCFilter))
	}

	//For each syncbox provided, Check SSH, Insert any new reports, and return all reports found in DB
	for _, s := range syncboxes {
		//Check SSH
		batch = sshDataAccess.GetMtrData_SpecificTimeframe(s, start, end)
		//Insert any new reports into the DB
		insertMtrReportsIntoDB(batch)

		//Select the matching reports from the DB
		if isFlagPassed("dc") {
			batch = sqlDataAccessor.SelectMtrReports_BySyncbox_DCAndTimeframe(s, start, end, DCFilter)
			fmt.Println("Reports found for "+s+" going to "+strings.ToUpper(DCFilter)+":", len(batch))

		} else {
			batch = sqlDataAccessor.SelectMtrReports_BySyncbox_Timeframe(s, start, end)
			fmt.Println("Reports found for "+s+":", len(batch))
		}
		mtrReports = append(mtrReports, batch...)
	}
	return mtrReports
}

// Check SSH, Insert new reports into DB, Select from DB.
// Retrieves all log files for the specified boxes at the time provided
func getMtrData_TargetTime(syncboxes []string, targetTime time.Duration, DCFilter string) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batch []dataObjects.MtrReport

	start := time.Now().UTC().Add(-(targetTime + (time.Minute * 3)))
	end := time.Now().UTC().Add(-(targetTime - (time.Minute * 3)))

	fmt.Println("Target Time:\t" +
		fmt.Sprint(start.Add(time.Minute*3).Format(time.ANSIC)))
	if isFlagPassed("dc") {
		fmt.Println("Data Center:\t" + strings.ToUpper(DCFilter))
	}

	//For each syncbox provided, Check SSH, Insert any new reports, and return all reports found in DB
	for _, s := range syncboxes {
		//Check SSH
		batch = sshDataAccess.GetMtrData_SpecificTimeframe(s, start, end)
		//Insert any new reports into the DB
		insertMtrReportsIntoDB(batch)

		//Select the matching reports from the DB
		if isFlagPassed("dc") {
			batch = sqlDataAccessor.SelectMtrReports_BySyncbox_DCAndTimeframe(s, start, end, DCFilter)
			fmt.Println("Reports found for "+s+" going to "+strings.ToUpper(DCFilter)+":", len(batch))
		} else {
			batch = sqlDataAccessor.SelectMtrReports_BySyncbox_Timeframe(s, start, end)
			fmt.Println("Reports found for "+s+":", len(batch))
		}
		mtrReports = append(mtrReports, batch...)
	}

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
	var syncboxes []string
	for _, s := range syncboxArgs {
		syncboxes = append(syncboxes, strings.ToLower(s))
	}
	return syncboxes, startTime, endTime, dcFilter, hostname
}

// Displays program details to console
func programDisplay() {
	fmt.Print("\n===============================================")
	fmt.Print(" Mtr Tools ")
	fmt.Println("===============================================")
	fmt.Println("Flags:")
	fmt.Println("\t-a\tRuns a sweep of ALL Syncboxes")
	fmt.Println("\t-start\tSpecifies a target search start time. Eg. 5h30m = 5 hours and 30 minutes ago")
	fmt.Println("\t-end\tSpecifies a target search end time. Eg. 0m = now, 0 minutes ago")
	fmt.Println("\t-p\tPrints results to the command line")
	fmt.Println("\t-pf\tPrint the results to a text file")
	fmt.Println("Syncboxes:", len(_SyncboxList))
	for i, s := range _SyncboxList {
		if i%7 == 0 {
			fmt.Print("\n" + s)
		} else {
			fmt.Print("\t" + s)
		}
	}
}

// Takes a slice of MTR Reports, checks if each is already in the DB, if not it inserts it
func insertMtrReportsIntoDB(mtrReports []dataObjects.MtrReport) {
	if len(mtrReports) > 0 {
		for _, r := range mtrReports {
			//Check if the Report already exists in the DB
			successfulInsert := sqlDataAccessor.InsertMtrReport(r)

			if !successfulInsert {
				fmt.Println("Error inserting ", r.ReportID)
			}
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
	//Set the SyncboxList equal to the DB list
	_SyncboxList = dbSyncboxList
	//Print count of SyncboxList
	//fmt.Println("Total Syncboxes: " + fmt.Sprint(len(_SyncboxList)) + "\n")
}

// Helper function to validate a timeframe
func validateTimeframe(startTime time.Duration, endTime time.Duration) bool {
	// Check for valid start and end times
	validTimes := false

	switch {
	// Check if start time is 5 minutes in the past
	case !time.Now().Add(-startTime).Before(time.Now().Add(-time.Minute * 4)):
		validTimes = false
		fmt.Println("Start time must be 5 minutes or more ago.")
	// Check if end time is after time.Now() time.Now()
	case !(time.Now().Add(-endTime).Before(time.Now()) || time.Now().Add(-endTime).Equal(time.Now())):
		validTimes = false
		fmt.Println("End time cannot be in the future")
	// Check if end time is before start time
	case !time.Now().Add(-endTime).After(time.Now().Add(-startTime)):
		validTimes = false
		fmt.Println("End time must be after start time.")
	// If none of these cases arise, times are valid
	default:
		validTimes = true
	}

	return validTimes
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

// Sets the values provided for the flags accepted by the program
func setFlags() (time.Duration, time.Duration, string, string) {
	var all bool
	flag.BoolVar(&all, "a", false, "Target ALL syncboxes")
	defaultTime := time.Since(time.Now())
	var startTime time.Duration
	flag.DurationVar(&startTime, "start", defaultTime, "Search timeframe start time")
	var endTime time.Duration
	flag.DurationVar(&endTime, "end", defaultTime, "Search timeframe end time")
	var printResult bool
	flag.BoolVar(&printResult, "p", false, "Print search results to command-line")
	var filterByDataCenter string
	flag.StringVar(&filterByDataCenter, "dc", "", "Filter search results by data center")
	var printToFile bool
	flag.BoolVar(&printToFile, "pf", false, "Print results to a text file")
	var hostname string
	flag.StringVar(&hostname, "host", "", "View reports involving the host name provided")

	flag.Parse()
	return startTime, endTime, filterByDataCenter, hostname
}

// Verifies if a flag has been provided
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
