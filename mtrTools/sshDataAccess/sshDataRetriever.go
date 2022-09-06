package sshDataAccess

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"mtrTools/dataObjects"

	"golang.org/x/crypto/ssh"
)

//Gets ALL mtrs in a specified syncbox's directory for a specified date
func GetSyncboxMtrReports(syncbox string, targetDate time.Time) ([]dataObjects.MtrReport, string) {
	var validatedMtrReports []dataObjects.MtrReport
	syncboxStatus := ""
	conn := connectToSSH()

	if conn != nil {
		defer conn.Close()
		mtrLogFilenames, err := getSyncboxLogFilenames(conn, syncbox, targetDate)
		if err != nil {
			syncboxStatus = "Inactive"
		} else {
			if len(mtrLogFilenames) > 0 {
				rawMtrData := getSyncboxMtrData(conn, syncbox, targetDate)
				//fmt.Println("Got Log Data...")
				tempMtrReports := parseSshDataIntoMtrReport(rawMtrData)
				//fmt.Println("Parsed data into reports...")
				validatedMtrReports = matchMtrDataWithFilenames(mtrLogFilenames, tempMtrReports)
				//fmt.Println("Validated Report ID...")
				if len(validatedMtrReports) > 0 {
					syncboxStatus = "Active"
				} else {
					syncboxStatus = "Firewall"
				}
			} else {
				syncboxStatus = "Inactive"
			}
		}

	} else {
		fmt.Println("Could not establish connection for " + syncbox)
	}

	return validatedMtrReports, syncboxStatus
}

//Gets ALL mtrs in a specified syncbox's directory that have a start time between two specified datetimes
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

func connectToSSH() *ssh.Client {
	var conn *ssh.Client
	var err error
	config := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// connect to ssh server
	conn, err = ssh.Dial("tcp", sshTargetHost, config)

	if err != nil {
		for i := 1; i < 4; i++ {
			time.Sleep(time.Second * 5)
			fmt.Println("Error connecting to SSH Server. Retry attempt", i, "...")
			conn, err = ssh.Dial("tcp", sshTargetHost, config)
			if conn != nil {
				break
			}
		}
	}
	return conn
}

//This sets up the ssh connection and runs the given command
func runClientCommand(conn *ssh.Client, command string) (string, error) {
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
