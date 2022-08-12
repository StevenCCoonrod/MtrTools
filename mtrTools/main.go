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

	//Update the SyncboxList []string
	updateSyncboxList()

	startTime, endTime, DCFilter := setFlags()
	syncboxes := flag.Args()

	if len(os.Args) > 1 {
		// Specific Window of Time Functions
		if isFlagPassed("start") && isFlagPassed("end") {
			if isFlagPassed("a") {
				// Specific Window of Time Functions on ALL boxes
				fmt.Println("Initiating Full Timeframe Sweep at ", time.Now().UTC())
				getMtrData_SpecificTimeframe(_SyncboxList, startTime, endTime, DCFilter)

			} else {
				// Specific Window of Time Functions on Specific boxes
				getMtrData_SpecificTimeframe(syncboxes, startTime, endTime, DCFilter)

			}
			//Specific Time Functions
		} else if isFlagPassed("start") && !isFlagPassed("end") {
			if isFlagPassed("a") {
				//Specific Time functions on ALL boxes
				getMtrData_SpecificTime(_SyncboxList, startTime, DCFilter)
			} else {
				//Specific Time functions on Specific boxes
				getMtrData_SpecificTime(syncboxes, startTime, DCFilter)
			}
			//No Time Frame Functions
		} else {
			if isFlagPassed("a") {
				//No Time Frame Functions on ALL boxes
				runMtrRetrievalCycle(DCFilter)
			} else {
				//No Time Frame Functions on Specific boxes
			}
		}
	} else {
		//No args given
		//Use this to target a problem box or method
		// reports := []string{"keye-2309-2022-08-04-14-55-da-mtr-catcher.log", "keye-2309-2022-08-04-14-57-dc-mtr-catcher.log"}
		// sqlDataAccessor.SelectMtrReportsByID(reports)
	}
}

//Retrieves ALL MTR logs for ALL syncboxes in the SyncboxList
func runMtrRetrievalCycle(DCFilter string) {
	fmt.Println("Initiating Full Sweep at ", time.Now().UTC())
	fmt.Println("============ Beginning Full MTR Sweep ============")
	for _, s := range _SyncboxList {
		var batch []string
		batch = append(batch, s)
		//mtrReports :=
		getMtrData_SpecificTimeframe(
			batch, time.Since(time.Now().UTC().AddDate(0, 0, -1)), time.Duration(0), DCFilter)
	}

	fmt.Println("============ MTR Sweep Completed ============")
}

//Check SSH, Insert new reports into DB, Select from DB.
//Retrieves all log files for the specified boxes between the two times provided
func getMtrData_SpecificTimeframe(syncboxes []string, startTime time.Duration, endTime time.Duration, DCFilter string) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batch []dataObjects.MtrReport

	start := time.Now().UTC().Add(-startTime)
	end := time.Now().UTC().Add(-endTime)
	fmt.Println("StartTime: " + fmt.Sprint(start) + "\nEndTime: " + fmt.Sprint(end))
	if isFlagPassed("dc") {
		fmt.Println("Data Center: " + strings.ToUpper(DCFilter))
	}

	//For each syncbox provided, Check SSH, Insert any new reports, and return all reports found in DB
	for _, s := range syncboxes {
		//Check SSH
		batch = sshDataAccess.GetMtrData_SpecificTimeframe(s, start, end)
		//Insert any new reports into the DB
		insertMtrReportsIntoDB(batch)

		//Select the matching reports from the DB
		if isFlagPassed("dc") {
			batch = sqlDataAccessor.SelectSyncboxMtrReportsByDCAndTimeframe(s, start, end, DCFilter)

		} else {
			batch = sqlDataAccessor.SelectMtrReportsByID(batch)
		}

		fmt.Println("Reports found for", s, ":", len(batch))

		if isFlagPassed("p") {
			for _, r := range batch {
				r.PrintReport()
			}
		}

		mtrReports = append(mtrReports, batch...)
	}

	return mtrReports
}

//Check SSH, Insert new reports into DB, Select from DB.
//Retrieves all log files for the specified boxes at the time provided
func getMtrData_SpecificTime(syncboxes []string, targetTime time.Duration, DCFilter string) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var batch []dataObjects.MtrReport
	start := time.Now().UTC().Add(-(targetTime + (time.Minute * 3)))
	end := time.Now().UTC().Add(-(targetTime - (time.Minute * 3)))
	//fmt.Println(start, " | ", end)
	//For each syncbox provided, Check SSH, Insert any new reports, and return all reports found in DB
	for _, s := range syncboxes {
		//Check SSH
		batch = sshDataAccess.GetMtrData_SpecificTimeframe(s, start, end)
		//fmt.Println(len(batch))
		//Insert any new reports into the DB
		insertMtrReportsIntoDB(batch)

		//Select the matching reports from the DB
		if isFlagPassed("dc") {
			batch = sqlDataAccessor.SelectSyncboxMtrReportsByDCAndTimeframe(s, start, end, DCFilter)

		} else {
			batch = sqlDataAccessor.SelectMtrReportsByID(batch)
		}
		//fmt.Println(len(batch))
		if isFlagPassed("p") {
			for _, r := range batch {
				r.PrintReport()
			}
		}
		fmt.Println(len(batch), "report(s) collected for", s)
		mtrReports = append(mtrReports, batch...)
	}

	return mtrReports
}

//Takes a slice of MTR Reports, checks if each is already in the DB, if not it inserts it
func insertMtrReportsIntoDB(mtrReports []dataObjects.MtrReport) {
	if len(mtrReports) > 0 {
		for _, r := range mtrReports {
			//Check if the Report already exists in the DB
			exists := sqlDataAccessor.CheckIfMtrReportExists(r.ReportID)
			//fmt.Println("Inserting " + r.ReportID)
			//If not, insert it
			if !exists {
				//fmt.Println("Inserting report: ", r.ReportID)
				successfulInsert := sqlDataAccessor.InsertMtrReport(r)
				// for _, h := range r.Hops {
				// 	sqlDataAccessor.InsertMtrHop(r.ReportID, h)

				// }
				if !successfulInsert {
					fmt.Println("There was an error inserting ", r.ReportID)
				}
			}
		}
	}
}

//Automatically compares the DB and SSH Server list of Syncboxes
//and adds any that aren't in the DB to the DB.
//Updates the SyncboxList
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
	fmt.Println("Total Syncboxes: " + fmt.Sprint(len(_SyncboxList)) + "\n")
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func setFlags() (time.Duration, time.Duration, string) {
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

	flag.Parse()
	return startTime, endTime, filterByDataCenter
}
