package sshDataAccess

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"mtrTools/dataObjects"

	"golang.org/x/crypto/ssh"
)

// Gets ALL mtrs in a specified syncbox's directory for a specified date
func GetSyncboxMtrReports(syncbox string, targetDate time.Time) ([]dataObjects.MtrReport, string) {
	var mtrReports []dataObjects.MtrReport
	syncboxStatus := ""
	conn := ConnectToSSH()

	if conn != nil {
		defer conn.Close()
		fmt.Println("Connected for " + syncbox)
		mtrLogFilenames := getSyncboxLogFilenames(conn, syncbox, targetDate)
		if len(mtrLogFilenames) > 0 {
			rawMtrData := getSyncboxMtrData(conn, syncbox, targetDate)
			//fmt.Println("Got Log Data...")
			if len(rawMtrData) > 0 {
				// mtrReports = ParseSshDataIntoMtrReport(rawMtrData, nil)

			} else {
				syncboxStatus = "Firewall"
			}
		} else {
			syncboxStatus = "Inactive"
		}
	} else {
		fmt.Println("Could not establish connection for " + syncbox)
	}

	return mtrReports, syncboxStatus
}

// Gets ALL mtrs in a specified syncbox's directory that have a start time between two specified datetimes
func GetMtrData_SpecificTimeframe(syncbox string, startTime time.Time, endTime time.Time) ([]dataObjects.MtrReport, string) {
	var mtrReports []dataObjects.MtrReport
	var unfilteredMtrReports []dataObjects.MtrReport
	var syncboxStatus string
	var reports []dataObjects.MtrReport
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {

		reports, syncboxStatus = GetSyncboxMtrReports(strings.ToLower(syncbox), d)

		unfilteredMtrReports = append(unfilteredMtrReports, reports...)
	}
	for _, r := range unfilteredMtrReports {
		if r.StartTime.After(startTime) && r.StartTime.Before(endTime) {
			mtrReports = append(mtrReports, r)
		}
	}
	//Get DB Reports within timeframe
	//Print the reports
	return mtrReports, syncboxStatus
}

// Step 1 in the MTR Data collection process.
// Retrieves the most recently added log file names found in each syncbox directory.
func getBatchSyncboxLogFilenames(conn *ssh.Client, syncboxes []string, targetDate time.Time) []string {
	validMonth := ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := ValidateDateField(fmt.Sprint(targetDate.Day()))
	var command string
	var dataReturned string
	filesToRetrieve := 20
	for _, s := range syncboxes {
		command = "cd " + BaseDirectory +
			fmt.Sprint(targetDate.Year()) + "/" +
			validMonth + "/" +
			validDay + "/" + strings.ToLower(s) +
			" && ls -t | head -" + fmt.Sprint(filesToRetrieve)
		dataReturned_1, err := RunClientCommand(conn, command)
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
func getBatchSyncboxMtrData(conn *ssh.Client, syncboxes []string, mtrLogFilenames []string, targetDate time.Time) ([]string, []string) {
	validMonth := ValidateDateField(fmt.Sprint(int32(targetDate.Month())))
	validDay := ValidateDateField(fmt.Sprint(targetDate.Day()))
	// var batchDataString string

	var rawReports []string
	// Target each Syncbox directory in this batch, build and run a command for each log file provided
	for _, s := range syncboxes {

		var command string
		var dataReturned string

		for _, l := range mtrLogFilenames {
			var err error
			// Check that the filename contains the syncbox name so that only the data of log files for this box is returned
			if strings.Contains(l, strings.ToLower(s)) {
				// Build a command targeting this specific log file in the target Syncboxes directory
				command = "cat " + BaseDirectory +
					fmt.Sprint(targetDate.Year()) + "/" +
					validMonth + "/" +
					validDay + "/" + strings.ToLower(s) + "/"
				command += l

				// Run the command
				dataReturned, err = runBatchMtrClientCommand(conn, command)
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

func ConnectToSSH() *ssh.Client {
	var conn *ssh.Client
	var err error
	config := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 20,
	}

	// connect to ssh server
	conn, err = ssh.Dial("tcp", sshTargetHost, config)

	if err != nil {
		for i := 1; i <= 5; i++ {
			if conn != nil {
				fmt.Println("Retry", i, "Successful.")
				break
			} else {
				time.Sleep(time.Second * 5)
				fmt.Println("Error connecting to SSH Server. Retry attempt", i, "...")
				conn, err = ssh.Dial("tcp", sshTargetHost, config)
				if err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	}
	return conn
}

// This sets up the ssh connection and runs the given command
func RunClientCommand(conn *ssh.Client, command string) (string, error) {
	var buff bytes.Buffer
	var err2 error
	if conn != nil {
		session, err := conn.NewSession()
		if err != nil {
			fmt.Println("Error beginning session on SSH Server.\n" + err.Error())
		}
		defer session.Close()

		session.Stdout = &buff

		if err2 = session.Run(command); err2 != nil {
			fmt.Println("Command:", command)
			fmt.Println("Error returned:", err2.Error())
		}
	}

	return buff.String(), err2
}
