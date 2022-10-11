package sshDataAccess

import (
	"fmt"
	"mtrTools/dataObjects"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var sshUser string = ""
var sshPassword string = ""
var sshTargetHost string = "master.syncbak.com:22"
var BaseDirectory string = "/var/log/syncbak/catcher-mtrs/"

// Retrieves a list of syncboxes from the current day's mtr directory
func GetSyncboxList() []string {

	date := time.Now()
	validMonth := ValidateDateField(fmt.Sprint(int32(date.Month())))
	validDay := ValidateDateField(fmt.Sprint(date.Day()))
	var syncboxList []string

	command := "ls " + BaseDirectory +
		fmt.Sprint(date.Year()) + "/" +
		validMonth + "/" + validDay + "/"

	conn := ConnectToSSH()
	data, err := RunClientCommand(conn, command)
	if err != nil {
		fmt.Println(err)
	}
	//defer conn.Close()

	tempSyncboxList := strings.Split(data, "\n")
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

// Gets the FILE NAMES of ALL logs in the specified date and syncbox directory
func getSyncboxLogFilenames(conn *ssh.Client, syncbox string, targetDate time.Time) []string {
	validMonth := ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := ValidateDateField(fmt.Sprint(targetDate.Day()))

	command1 := "ls " + BaseDirectory +
		fmt.Sprint(targetDate.Year()) + "/" +
		validMonth + "/" +
		validDay + "/" +
		syncbox + "/"
	dataReturned_1, err := RunClientCommand(conn, command1)
	if err != nil {
		if strings.Contains(err.Error(), "Process exited with status 1") {
			fmt.Println("No log files found in the " + syncbox + " directory.")
		} else {
			fmt.Println("Error running command on SSH Server.\n" + err.Error())
		}
	}
	return strings.Split(dataReturned_1, "\n")
}

// Gets the LOG FILE DATA of ALL logs in the specified date and syncbox directory
func getSyncboxMtrData(conn *ssh.Client, syncbox string, targetDate time.Time) string {
	validMonth := ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := ValidateDateField(fmt.Sprint(targetDate.Day()))

	command2 := "cat " + BaseDirectory +
		fmt.Sprint(targetDate.Year()) + "/" +
		validMonth + "/" +
		validDay + "/" +
		syncbox + "/" + "*.log"

	dataReturned_2, err := RunClientCommand(conn, command2)
	if err != nil {
		if strings.Contains(err.Error(), "Process exited with status 1") {
			fmt.Println("Error retrieving log data in the " + syncbox + " directory.")
		} else {
			fmt.Println("Error running command on SSH Server.\n" + err.Error())
		}
	}
	return dataReturned_2
}

// Parses raw MTR data into a slice of MtrReports
func ParseSshDataIntoMtrReport(rawData []string, LogFilenames []string) []dataObjects.MtrReport {

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
			currentReportsStartTime = time.Date(ParseStringToInt(logfilenameFields[2]),
				time.Month(ParseStringToInt(logfilenameFields[3])),
				ParseStringToInt(logfilenameFields[4]),
				ParseStringToInt(logfilenameFields[5]),
				ParseStringToInt(logfilenameFields[6]), 0, 0, time.UTC)
			currentReportsTargetDC = logfilenameFields[7]

		} else {
			currentReportsHost = logfilenameFields[0]
			currentReportsStartTime = time.Date(ParseStringToInt(logfilenameFields[1]),
				time.Month(ParseStringToInt(logfilenameFields[2])),
				ParseStringToInt(logfilenameFields[3]),
				ParseStringToInt(logfilenameFields[4]),
				ParseStringToInt(logfilenameFields[5]), 0, 0, time.UTC)
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

		hop.HopNumber = ParseStringToInt(hn)
		if len(f) > 1 {
			hop.Hostname = f[1]
		}
		if len(f) > 2 {
			pl := strings.Replace(f[2], "%", "", 1)
			hop.PacketLoss = ParseStringToFloat32(pl)
		}
		if len(f) > 3 {
			hop.PacketsSent = ParseStringToInt(f[3])
		}
		if len(f) > 4 {
			hop.LastPing = ParseStringToFloat32(f[4])
		}
		if len(f) > 5 {
			hop.AveragePing = ParseStringToFloat32(f[5])
		}
		if len(f) > 6 {
			hop.BestPing = ParseStringToFloat32(f[6])
		}
		if len(f) > 7 {
			hop.WorstPing = ParseStringToFloat32(f[7])
		}
		if len(f) > 8 {
			hop.StdDev = ParseStringToFloat32(f[8])
		}
		mtrReport.Hops = append(mtrReport.Hops, hop)
	}
	return mtrReport
}

// Helper method to provide valid date fields for mtr directories ("07" instead of "7")
func ValidateDateField(dateField string) string {
	if len(dateField) == 1 {
		dateField = "0" + dateField
	}
	return dateField
}

// Helper method to parse strings into a float32
func ParseStringToFloat32(s string) float32 {
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
func ParseStringToInt(s string) int {
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
