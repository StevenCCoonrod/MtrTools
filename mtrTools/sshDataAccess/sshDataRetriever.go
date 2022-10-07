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
				mtrReports = ParseSshDataIntoMtrReport(rawMtrData, nil)

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
