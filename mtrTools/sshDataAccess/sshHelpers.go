package sshDataAccess

import (
	"fmt"
	"mtrTools/dataObjects"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var sshUser string = ""
var sshPassword string = ""
var sshTargetHost string = "master3.syncbak.com:22"
var baseDirectory string = "/var/log/syncbak/catcher-mtrs/"

// Retrieves a list of syncboxes from the current day's mtr directory
func GetSyncboxList() ([]string, error) {

	date := time.Now()
	validMonth := validateDateField(fmt.Sprint(int32(date.Month())))
	validDay := validateDateField(fmt.Sprint(date.Day()))
	var syncboxList []string

	command := "ls " + baseDirectory +
		fmt.Sprint(date.Year()) + "/" +
		validMonth + "/" + validDay + "/"

	conn := connectToSSH()
	data, err := runClientCommand(conn, command)
	if err == nil {
		defer conn.Close()

		tempSyncboxList := strings.Split(data, "\n")
		for _, s := range tempSyncboxList {
			if strings.Contains(s, "-2309") {
				syncboxList = append(syncboxList, s)
			}
		}
	}

	return syncboxList, err
}

// Gets the FILE NAMES of ALL logs in the specified date and syncbox directory
func getSyncboxLogFilenames(conn *ssh.Client, syncbox string, targetDate time.Time) ([]string, error) {
	validMonth := validateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := validateDateField(fmt.Sprint(targetDate.Day()))

	command1 := "ls " + baseDirectory +
		fmt.Sprint(targetDate.Year()) + "/" +
		validMonth + "/" +
		validDay + "/" +
		syncbox + "/"
	dataReturned_1, err := runClientCommand(conn, command1)
	// if err != nil {
	// 	if strings.Contains(err.Error(), "Process exited with status 1") {
	// 		fmt.Println("No log files found in the " + syncbox + " directory.")
	// 	} else {
	// 		fmt.Println("Error running command on SSH Server.\n" + err.Error())
	// 	}
	// }
	return strings.Split(dataReturned_1, "\n"), err
}

// Gets the LOG FILE DATA of ALL logs in the specified date and syncbox directory
func getSyncboxMtrData(conn *ssh.Client, syncbox string, targetDate time.Time) string {
	validMonth := validateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := validateDateField(fmt.Sprint(targetDate.Day()))

	command2 := "cat " + baseDirectory +
		fmt.Sprint(targetDate.Year()) + "/" +
		validMonth + "/" +
		validDay + "/" +
		syncbox + "/" + "*.log"

	dataReturned_2, err := runClientCommand(conn, command2)
	if err != nil {
		if strings.Contains(err.Error(), "Process exited with status 1") {
			fmt.Println("Error retrieving log data in the " + syncbox + " directory.")
		} else {
			fmt.Println("Error running command on SSH Server.\n" + err.Error())
		}
	}
	return dataReturned_2
}

// Compares a list of mtr filenames with a list of raw mtr data, and assigns matching filenames to the corresponding report's ID field
func matchMtrDataWithFilenames(mtrLogFilenames []string, tempMtrReports []dataObjects.MtrReport) []dataObjects.MtrReport {
	var validatedMtrReports []dataObjects.MtrReport
	for _, l := range mtrLogFilenames {
		if l != "" {
			//Split each line on the "-" and parse the fields
			f := strings.Split(l, "-")
			dateYear := ParseStringToInt(f[2])
			dateMonth := time.Month(ParseStringToInt(f[3]))
			dateDay := ParseStringToInt(f[4])
			dateHour := ParseStringToInt(f[5])
			dateMinute := ParseStringToInt(f[6])
			dataCenter := f[7]

			//Parse this into a time.Time object
			logFileDateTime := time.Date(dateYear, dateMonth, dateDay, dateHour, dateMinute, 0, 0, &time.Location{})

			//Match the parsed datetime from the filename
			//with the corresponding report in the mtr list
			for _, r := range tempMtrReports {
				id := strings.ReplaceAll(l, " ", "-")

				if r.StartTime.Year() == logFileDateTime.Year() &&
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

// Parses raw MTR data into a slice of MtrReports
func parseSshDataIntoMtrReport(rawData string) []dataObjects.MtrReport {

	//Create the Report array to hold all the retrieved mtr Reports
	var mtrReports []dataObjects.MtrReport

	//rawData should contain ALL mtr data for ALL mtr log files in a specific syncbox directory
	rawMtrData := strings.Split(rawData, "Start: ")
	if len(rawMtrData) > 1 {
		//At this point, the full data string should be split back into
		//strings containing the data for each individual log file

		//Loop through each raw report string and parse into an MtrReport object
		for _, m := range rawMtrData {

			if m != "" && !strings.Contains(m, "<!") {
				//Create new mtrReport
				mtrReport := dataObjects.MtrReport{}
				//Split data into lines
				lines := strings.Split(m, "\n")
				//Iterate through each line in the data
				for i, l := range lines {

					//fmt.Println(l)

					//If its the first line, parse the StartTime datetime
					if i == 0 {
						p := strings.TrimSpace(l)
						startTime, err := time.Parse(time.ANSIC, p)
						if err != nil {
							fmt.Println("There was a problem parsing the mtr data.\n" + m + "\n" + err.Error())
						}
						mtrReport.StartTime = startTime
						//If its the second line, remove everything that isn't the Syncbox ID
					} else if i == 1 {
						s := strings.Replace(l, "HOST: ", "", 1)
						s = strings.Split(s, ".")[0]
						mtrReport.SyncboxID = s
						//Otherwise, each line is a hop in the traceroute
					} else {
						if l != "" {
							//Create new hop
							hop := dataObjects.MtrHop{}
							//Split the line by fields and parse a new hop
							f := strings.Fields(l)

							//Painful way of checking that fields are not null
							if len(f) > 0 {
								hn := f[0]
								hn = strings.Replace(hn, ".", "", 1) // Why? random mtr's threw errors because the hop number (f[0]) only had a "." instead of ".|--"
								hn = strings.Replace(hn, "|--", "", 1)

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
						}
					}
				}

				lastHopHost := "No Hops in Report"
				if len(mtrReport.Hops) > 0 {
					//Verify the data center using the final hop hostname
					lastHopHost = mtrReport.Hops[len(mtrReport.Hops)-1].Hostname
				}

				if len(mtrReport.Hops) >= 1 && strings.Contains(lastHopHost, "util") {

					lastHopDataCenter := strings.Replace(lastHopHost, "util", "", 1)
					lastHopDataCenter = strings.Replace(lastHopDataCenter, "eqnx", "", 1)
					mtrReport.DataCenter = lastHopDataCenter
				} else {
					mtrReport.DataCenter = "na"
				}

				mtrReports = append(mtrReports, mtrReport)
			}
		}
	}

	return mtrReports
}

// Helper method to provide valid date fields for mtr directories ("07" instead of "7")
func validateDateField(dateField string) string {
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
