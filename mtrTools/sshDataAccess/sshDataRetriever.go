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
func GetSyncboxMtrReports(syncbox string, targetDate time.Time) []dataObjects.MtrReport {

	mtrLogFilenames := getSyncboxLogFilenames(syncbox, targetDate)
	rawMtrData := getSyncboxMtrData(syncbox, targetDate)
	tempMtrReports := parseSshDataIntoMtrReport(rawMtrData)
	validatedMtrReports := matchMtrDataWithFilenames(mtrLogFilenames, tempMtrReports)

	return validatedMtrReports
}

//Gets ALL mtrs in a specified syncbox's directory that have a start time between two specified datetimes
func GetMtrData_SpecificTimeframe(syncbox string, startTime time.Time, endTime time.Time) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var unfilteredMtrReports []dataObjects.MtrReport
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {

		reports := GetSyncboxMtrReports(strings.ToLower(syncbox), d)

		unfilteredMtrReports = append(unfilteredMtrReports, reports...)
	}
	for _, r := range unfilteredMtrReports {
		if r.StartTime.After(startTime) && r.StartTime.Before(endTime) {
			mtrReports = append(mtrReports, r)
		}
	}
	//Get DB Reports within timeframe
	//Print the reports
	return mtrReports
}

////Gets ALL mtrs in a specified syncbox's directory that have a start time within 5 minutes of the specified time
func GetMtrData_SpecificTime(syncbox string, targetTime time.Time) []dataObjects.MtrReport {
	var mtrReports []dataObjects.MtrReport
	var unfilteredMtrReports []dataObjects.MtrReport
	startTime := targetTime.Add(-time.Minute * 2)
	endTime := targetTime.Add(time.Minute * 2)
	for d := targetTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {

		reports := GetSyncboxMtrReports(strings.ToLower(syncbox), d)

		unfilteredMtrReports = append(unfilteredMtrReports, reports...)
	}
	for _, r := range unfilteredMtrReports {
		if r.StartTime.After(startTime) && r.StartTime.Before(endTime) {
			mtrReports = append(mtrReports, r)
		}
	}
	//Get DB Reports within timeframe
	//Print the reports
	return mtrReports
}

//This sets up the ssh connection and runs the given command
func runClientCommand(command string) (string, error) {

	//fmt.Println(command) //Not needed, used while testing

	config := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// connect to ssh server
	conn, err := ssh.Dial("tcp", sshTargetHost, config)
	if err != nil {
		fmt.Println("Error connecting to SSH Server.\n" + err.Error())
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		fmt.Println("Error beginning session on SSH Server.\n" + err.Error())
	}
	defer session.Close()

	var buff bytes.Buffer
	session.Stdout = &buff
	if err := session.Run(command); err != nil {
		fmt.Println("Error running command on SSH Server.\n" + err.Error())
	}
	return buff.String(), err
}
