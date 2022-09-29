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
var sshTargetHost string = "master3.syncbak.com:22"
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
	defer conn.Close()

	tempSyncboxList := strings.Split(data, "\n")
	for _, s := range tempSyncboxList {
		if strings.Contains(s, "-2309") {
			syncboxList = append(syncboxList, s)
		}
	}

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
func ParseSshDataIntoMtrReport(rawData string) []dataObjects.MtrReport {

	//Create the Report array to hold all the retrieved mtr Reports
	var mtrReports []dataObjects.MtrReport

	//rawData should contain ALL mtr data for ALL mtr log files in a specific syncbox directory
	rawMtrData := strings.Split(rawData, "Start: ")
	if len(rawMtrData) > 1 {
		//At this point, the full data string should be split back into
		//strings containing the data for each individual log file

		//Loop through each raw report string and parse into an MtrReport object
		for _, m := range rawMtrData {

			if m != "" && !strings.Contains(m, "<") {
				//Create new mtrReport
				mtrReport := dataObjects.MtrReport{}
				//Split data into lines
				lines := strings.Split(m, "\n")
				//Iterate through each line in the data
				for i, l := range lines {

					//If its the first line, parse the StartTime datetime
					if i == 0 {
						p := strings.TrimSpace(l)

						startTime, err := time.Parse(time.ANSIC, p)
						if err != nil {
							fmt.Println("There was a problem parsing the mtr data.\n" + m + "\n" + err.Error())
						} else {
							mtrReport.StartTime = startTime
						}

						//If its the second line, remove everything that isn't the Syncbox ID
					} else if i == 1 {
						s := strings.Replace(l, "HOST: ", "", 1)
						if strings.Contains(s, ".") {
							s = strings.Split(s, ".")[0]
						} else {
							s = strings.Split(s, " ")[0]
						}

						mtrReport.SyncboxID = strings.ToLower(s)
						//Otherwise, each line is a hop in the traceroute
					} else {
						if l != "" {
							//Create new hop
							hop := dataObjects.MtrHop{}
							//Split the line by fields and parse a new hop
							f := strings.Fields(l)

							//Painful way of checking that fields are not null
							if len(f) > 0 {
								var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-z0-9 ]+`)
								hn := nonAlphanumericRegex.ReplaceAllString(f[0], "")
								// hn = strings.Replace(hn, ".", "", 1) // Why? random mtr's threw errors because the hop number (f[0]) only had a "." instead of ".|--"
								// hn = strings.Replace(hn, "|--", "", 1)

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
