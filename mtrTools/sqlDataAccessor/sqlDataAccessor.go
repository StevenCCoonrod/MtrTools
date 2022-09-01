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

var server = "localhost"

// var server = "localhost\\MSSQLSERVER02"

var port = 1433
var user = ""
var password = ""
var database = "NetopsToolsDB"

var db *sql.DB

//================== INSERT STATEMENTS ===================\\

// Inserts a Syncbox into the DB
func InsertSyncbox(syncbox string) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	db := getDBConnection()

	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println(err.Error())
	}

	dataReturned, err := db.QueryContext(ctx, "sp_InsertSyncbox", syncbox)
	if err != nil {
		fmt.Println(err)
	}
	dataReturned.Close()
	db.Close()
}

// Inserts an SSH MtrReport into the DB
func InsertMtrReports(mtrReports []dataObjects.MtrReport) bool {

	successfulInsert := false
	db := getDBConnection()
	for _, report := range mtrReports {
		reportExists, ctx, cancel := CheckIfMtrReportExists(db, report.ReportID)
		defer cancel()
		defer db.Close()
		if !reportExists {
			//Insert the Report
			_, err := db.ExecContext(ctx, "sp_InsertMtrReport", report.ReportID, report.SyncboxID, report.StartTime, report.DataCenter)

			if err != nil {
				fmt.Println("Error inserting report. ", err.Error())
				successfulInsert = false
			} else {
				successfulInsert = true
			}
			//Insert the report's hops
			if successfulInsert {
				for _, h := range report.Hops {

					_, err := db.ExecContext(ctx, "sp_InsertMtrHop", report.ReportID, h.HopNumber, h.Hostname,
						h.PacketLoss, h.PacketsSent, h.LastPing, h.AveragePing, h.BestPing, h.WorstPing, h.StdDev)

					if err != nil {
						fmt.Println("Error inserting hop", h.HopNumber, "for", report.ReportID, "\n", err.Error())
						successfulInsert = false
						break
					} else {
						successfulInsert = true
					}
				}
			}
		} else {
			successfulInsert = true
		}
	}

	return successfulInsert
}

//================== SELECT STATEMENTS ===================\\

// Selects all Syncboxes in the DB
func SelectAllSyncboxes() []string {

	var syncboxList []string
	var err error
	db := getDBConnection()
	ctx := context.Background()

	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println(err.Error())
	}

	dataReturned, err := db.QueryContext(ctx, "sp_SelectAllSyncboxes")
	if err != nil {
		fmt.Println(err)
	} else {
		for dataReturned.Next() {
			var syncbox string
			err := dataReturned.Scan(&syncbox)
			if err != nil {
				fmt.Println(err)
			}
			syncboxList = append(syncboxList, syncbox)
		}
		dataReturned.Close()
	}
	db.Close()
	return syncboxList
}

// Takes a batch of MtrReports, Selects them from the DB, and parses them back into MtrReports
func SelectMtrReportsByID(reports []dataObjects.MtrReport) []dataObjects.MtrReport {

	var dataReturned *sql.Rows
	var reportsReturned []dataObjects.MtrReport
	var err error
	db := getDBConnection()
	ctx := context.Background()

	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println(err.Error())
	}
	for _, r := range reports {

		dataReturned, err = db.QueryContext(ctx, "sp_SelectMtrReportByID", r.ReportID)
		if err != nil {
			fmt.Println("Error selecting mtr report. ", err.Error())
		} else {
			parsedReport := parseSqlSingleReportDataIntoReport(dataReturned)
			dataReturned.Close()
			reportsReturned = append(reportsReturned, parsedReport)
		}

	}
	db.Close()
	return reportsReturned
}

// Returns all DB Reports for a specified syncbox, between two datetimes
func SelectMtrReports_BySyncbox_Timeframe(syncbox string, startTime time.Time, endTime time.Time) []dataObjects.MtrReport {

	var err error
	db := getDBConnection()
	ctx := context.Background()

	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println(err1.Error())
		return nil
	}

	var dataReturned *sql.Rows
	var reportsReturned []dataObjects.MtrReport
	defer db.Close()
	dataReturned, err = db.QueryContext(ctx, "sp_SelectMtrs_BySyncbox_WithinRange", syncbox, startTime, endTime)
	if err != nil {
		fmt.Println("Error selecting mtr report. ", err.Error())
	}

	reportsReturned = append(reportsReturned, parseSqlMultipleReportDataIntoReports(dataReturned)...)

	dataReturned.Close()
	return reportsReturned
}

// Returns all DB Reports for a specified syncbox, between two datetimes, targeting a specified data center
func SelectMtrReports_BySyncbox_DCAndTimeframe(syncbox string, startTime time.Time, endTime time.Time, datacenter string) []dataObjects.MtrReport {

	var err error
	db := getDBConnection()
	ctx := context.Background()

	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println(err.Error())
	}

	var dataReturned *sql.Rows
	var reportsReturned []dataObjects.MtrReport
	defer db.Close()
	dataReturned, err = db.QueryContext(ctx, "sp_SelectMtrs_BySyncbox_DCAndTimeframe", syncbox, startTime, endTime, strings.ToLower(datacenter))
	if err != nil {
		fmt.Println("Error selecting mtr report. ", err.Error())
	}

	reportsReturned = append(reportsReturned, parseSqlMultipleReportDataIntoReports(dataReturned)...)

	dataReturned.Close()

	return reportsReturned
}

// Returns all DB Reports for a specified syncbox, between two datetimes
func SelectMtrReports_ByHostname(hostname string) []dataObjects.MtrReport {
	var err error
	db := getDBConnection()
	ctx := context.Background()

	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println(err.Error())
	}

	var dataReturned *sql.Rows
	var reportsReturned []dataObjects.MtrReport
	defer db.Close()
	dataReturned, err = db.QueryContext(ctx, "sp_SelectMtrs_ByHostname", hostname)
	if err != nil {
		fmt.Println("Error selecting reports by host name. ", err.Error())
	}

	reportsReturned = append(reportsReturned, parseSqlMultipleReportDataIntoReports(dataReturned)...)

	dataReturned.Close()

	return reportsReturned
}

//======================= HELPER METHODS =======================\\

// Parses an individual report from the DB into an MtrReport
func parseSqlSingleReportDataIntoReport(sqlRowData *sql.Rows) dataObjects.MtrReport {

	var reportID, syncboxID, dataCenter, hostName *string
	var startTime *time.Time
	var hopID, hopNumber, packetsSent *int
	var packetLoss, last, avg, best, worst, stdDev *float32
	var report dataObjects.MtrReport

	for sqlRowData.Next() {
		if err := sqlRowData.Scan(&reportID, &syncboxID, &startTime, &dataCenter, &hopID, &hopNumber,
			&hostName, &packetLoss, &packetsSent, &last, &avg, &best, &worst, &stdDev); err != nil {
			fmt.Println(err.Error())
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

// Parses multiple reports from the DB into MtrReports
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
			fmt.Println(err.Error())
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

// Establishes DB connection and context
func getDBConnection() *sql.DB {
	var err error
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		fmt.Println("Error creating connection pool. \n" + err.Error())
	}
	return db
}

// Checks if a Report already exists in the DB
func CheckIfMtrReportExists(db *sql.DB, mtrReportID string) (bool, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	//defer cancel()

	var err error
	mtrExists := false
	err1 := db.PingContext(ctx)
	if err1 != nil {
		fmt.Println("Error pinging context\n", err.Error())
	}

	dataReturned := db.QueryRow("sp_CheckIfMtrReportExists", mtrReportID)
	if err != nil {
		fmt.Println("Error checking for Report: ", mtrReportID, "\n", err.Error())
		//dataReturned.Close()
	} else {
		var reportID *string
		if err := dataReturned.Scan(&reportID); err != nil {
			if strings.Contains(err.Error(), "no rows in result set") {

			} else {
				fmt.Println("Error scanning data for", reportID, err.Error())
			}
		}
		//dataReturned.Close()
		if reportID == nil {
			mtrExists = false
		} else {
			mtrExists = true
		}
	}
	return mtrExists, ctx, cancel
}
