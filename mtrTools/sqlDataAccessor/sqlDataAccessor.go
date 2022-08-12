package sqlDataAccessor

import (
	"context"
	"database/sql"
	"fmt"
	"mtrTools/dataObjects"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

//var server = "localhost"
var server = "localhost"
var port = 1433
var user = "stevec"
var password = "MoM0H@$ha$hin"
var database = "NetopsToolsDB"

var db *sql.DB

//================== INSERT STATEMENTS ===================\\

//Inserts a Syncbox into the DB
func InsertSyncbox(syncbox string) {

	db, ctx := getDBConnection()

	dataReturned, err := db.QueryContext(ctx, "sp_InsertSyncbox", syncbox)
	if err != nil {
		fmt.Println(err)
	}
	dataReturned.Close()
}

//Inserts an SSH MtrReport into the DB
func InsertMtrReport(mtrReport dataObjects.MtrReport) bool {

	successfulInsert := false
	db, ctx := getDBConnection()

	//Insert the Report
	dataReturned, err := db.QueryContext(ctx, "sp_InsertMtrReport", mtrReport.ReportID, mtrReport.SyncboxID, mtrReport.StartTime, mtrReport.DataCenter)
	if err != nil {
		dataReturned.Close()
		db.Close()
		fmt.Println("Error inserting report. ", err.Error())
		successfulInsert = false
	} else {
		successfulInsert = true
	}
	defer dataReturned.Close()
	defer db.Close()
	//Insert the report's hops
	if successfulInsert {
		for _, h := range mtrReport.Hops {
			db, ctx = getDBConnection()
			dataReturned, err := db.QueryContext(ctx, "sp_InsertMtrHop", mtrReport.ReportID, h.HopNumber, h.Hostname,
				h.PacketLoss, h.PacketsSent, h.LastPing, h.AveragePing, h.BestPing,
				h.WorstPing, h.StdDev)
			if err != nil {
				dataReturned.Close()
				db.Close()
				fmt.Println("Error inserting hop. \n", err.Error())
				successfulInsert = false
				break
			} else {
				successfulInsert = true
			}
			defer dataReturned.Close()
			defer db.Close()
		}

	}

	return successfulInsert
}

//================== SELECT STATEMENTS ===================\\

//Selects all Syncboxes in the DB
func SelectAllSyncboxes() []string {

	var syncboxList []string
	db, ctx := getDBConnection()

	dataReturned, err := db.QueryContext(ctx, "sp_SelectAllSyncboxes")
	if err != nil {
		fmt.Println(err)
	}
	for dataReturned.Next() {
		var syncbox string
		err := dataReturned.Scan(&syncbox)
		if err != nil {
			panic(err)
		}
		syncboxList = append(syncboxList, syncbox)
	}
	dataReturned.Close()
	return syncboxList
}

//Takes a batch of SSH MtrReports, Selects them from the DB, and parses them back into MtrReports
func SelectMtrReportsByID(reports []dataObjects.MtrReport) []dataObjects.MtrReport {

	var dataReturned *sql.Rows
	var err error
	var reportsReturned []dataObjects.MtrReport
	for _, r := range reports {
		db, ctx := getDBConnection()
		dataReturned, err = db.QueryContext(ctx, "sp_SelectMtrReportByID", r.ReportID)
		if err != nil {
			fmt.Println("Error selecting mtr report. ", err.Error())
		}

		parsedReport := parseSqlSingleReportDataIntoReport(dataReturned)
		dataReturned.Close()
		reportsReturned = append(reportsReturned, parsedReport)
		db.Close()
	}
	return reportsReturned
}

//Returns all DB Reports for a specified syncbox, between two datetimes, targeting a specified data center
func SelectSyncboxMtrReportsByDCAndTimeframe(syncbox string, startTime time.Time, endTime time.Time, datacenter string) []dataObjects.MtrReport {

	var err error
	db, ctx := getDBConnection()
	var dataReturned *sql.Rows
	var reportsReturned []dataObjects.MtrReport
	defer db.Close()
	dataReturned, err = db.QueryContext(ctx, "sp_SelectSyncboxMtrsByDCAndTimeframe", syncbox, startTime, endTime, strings.ToLower(datacenter))
	if err != nil {
		fmt.Println("Error selecting mtr report. ", err.Error())
	}

	reportsReturned = append(reportsReturned, parseSqlMultipleReportDataIntoReports(dataReturned)...)

	dataReturned.Close()

	return reportsReturned
}

//======================= HELPER METHODS =======================\\

//Parses an individual report from the DB into an MtrReport
func parseSqlSingleReportDataIntoReport(sqlRowData *sql.Rows) dataObjects.MtrReport {

	var reportID, syncboxID, dataCenter, hostName *string
	var startTime *time.Time
	var hopID, hopNumber, packetsSent *int
	var packetLoss, last, avg, best, worst, stdDev *float32
	var report dataObjects.MtrReport

	for sqlRowData.Next() {
		if err := sqlRowData.Scan(&reportID, &syncboxID, &startTime, &dataCenter, &hopID, &hopNumber,
			&hostName, &packetLoss, &packetsSent, &last, &avg, &best, &worst, &stdDev); err != nil {
			panic(err.Error())
		} else {
			if *reportID != report.ReportID {

				report = dataObjects.MtrReport{ReportID: *reportID, SyncboxID: *syncboxID, StartTime: *startTime, DataCenter: *dataCenter}
				hop := dataObjects.MtrHop{HopID: *hopID, HopNumber: *hopNumber, Hostname: *hostName,
					PacketLoss: *packetLoss, PacketsSent: *packetsSent, LastPing: *last, AveragePing: *avg,
					BestPing: *best, WorstPing: *worst, StdDev: *stdDev}
				report.Hops = append(report.Hops, hop)
			} else {
				hop := dataObjects.MtrHop{HopID: *hopID, HopNumber: *hopNumber, Hostname: *hostName,
					PacketLoss: *packetLoss, PacketsSent: *packetsSent, LastPing: *last, AveragePing: *avg,
					BestPing: *best, WorstPing: *worst, StdDev: *stdDev}

				report.Hops = append(report.Hops, hop)

			}
		}

	}

	return report
}

//Parses multiple reports from the DB into MtrReports
func parseSqlMultipleReportDataIntoReports(sqlRowData *sql.Rows) []dataObjects.MtrReport {
	var reports []dataObjects.MtrReport
	var reportID, syncboxID, dataCenter, hostName *string
	var startTime *time.Time
	var hopID, hopNumber, packetsSent *int
	var packetLoss, last, avg, best, worst, stdDev *float32
	var report dataObjects.MtrReport
	for sqlRowData.Next() {
		if err := sqlRowData.Scan(&reportID, &syncboxID, &startTime, &dataCenter, &hopID, &hopNumber,
			&hostName, &packetLoss, &packetsSent, &last, &avg, &best, &worst, &stdDev); err != nil {
			panic(err.Error())
		} else {
			if *reportID != report.ReportID {
				if report.ReportID != "" {
					reports = append(reports, report)
				}

				report = dataObjects.MtrReport{ReportID: *reportID, SyncboxID: *syncboxID, StartTime: *startTime, DataCenter: *dataCenter}
				hop := dataObjects.MtrHop{HopID: *hopID, HopNumber: *hopNumber, Hostname: *hostName,
					PacketLoss: *packetLoss, PacketsSent: *packetsSent, LastPing: *last, AveragePing: *avg,
					BestPing: *best, WorstPing: *worst, StdDev: *stdDev}
				report.Hops = append(report.Hops, hop)
			} else {
				hop := dataObjects.MtrHop{HopID: *hopID, HopNumber: *hopNumber, Hostname: *hostName,
					PacketLoss: *packetLoss, PacketsSent: *packetsSent, LastPing: *last, AveragePing: *avg,
					BestPing: *best, WorstPing: *worst, StdDev: *stdDev}

				report.Hops = append(report.Hops, hop)
			}
		}
	}

	return reports
}

//Establishes DB connection and context
func getDBConnection() (*sql.DB, context.Context) {
	var err error
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool. \n" + err.Error())
	}

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		fmt.Println("Error pinging context. \n", err.Error())
	}
	return db, ctx
}

//Checks if a Report already exists in the DB
func CheckIfMtrReportExists(mtrReportID string) bool {
	var err error
	var mtrExists bool
	db, ctx := getDBConnection()

	dataReturned, err := db.QueryContext(ctx, "sp_CheckIfMtrReportExists", mtrReportID)
	if err != nil {
		fmt.Println("Error checking for mtr report. ", err.Error())
	}

	db.Close()

	var reportID *string
	for dataReturned.Next() {
		if err := dataReturned.Scan(&reportID); err != nil {
			dataReturned.Close()
			panic(err.Error())
		}
	}
	dataReturned.Close()
	if reportID == nil {
		mtrExists = false
	} else {
		mtrExists = true
	}

	return mtrExists
}
