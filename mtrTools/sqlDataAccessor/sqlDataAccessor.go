package sqlDataAccessor

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mtrTools/dataObjects"
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

//Test Method. Makes a simple connection to the DB
func ConnectToDB() {
	var err error

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
		server, user, password, port, database)

	db, err = sql.Open(database, connString)
	if err != nil {
		panic("Error creating connection pool" + err.Error())
	}
	log.Printf("Connected!\n")

	defer db.Close()
}

//================== INSERT STATEMENTS ===================\\

//Inserts a Syncbox into the DB
func InsertSyncbox(syncbox string) {
	var err error

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool \n" + err.Error())
	}
	defer db.Close()

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		panic(err)
	}

	dataReturned, err := db.QueryContext(ctx, "sp_InsertSyncbox", syncbox)
	if err != nil {
		fmt.Println(err)
	}
	dataReturned.Close()
	fmt.Println(dataReturned)
}

//Inserts an MTR Report into the DB
func InsertMtrReport(mtrReport dataObjects.MtrReport) {
	var err error

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool \n" + err.Error())
	}

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		fmt.Println("Error inserting report. ", err.Error())
	}

	dataReturned, err := db.QueryContext(ctx, "sp_InsertMtrReport", mtrReport.ReportID, mtrReport.SyncboxID, mtrReport.StartTime, mtrReport.DataCenter)
	if err != nil {
		dataReturned.Close()
		db.Close()
		fmt.Println("Error inserting report. ", err.Error())
	}
	dataReturned.Close()
	db.Close()
}

//Inserts an MTR Report hop into the DB
func InsertMtrHop(reportID string, hop dataObjects.MtrHop) {
	var err error

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool \n" + err.Error())
	}

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		fmt.Println("Error inserting hop. ", err.Error())
	}

	dataReturned, err := db.QueryContext(ctx, "sp_InsertMtrHop", reportID, hop.HopNumber, hop.Hostname,
		hop.PacketLoss, hop.PacketsSent, hop.LastPing, hop.AveragePing, hop.BestPing,
		hop.WorstPing, hop.StdDev)
	if err != nil {
		dataReturned.Close()
		db.Close()
		fmt.Println("Error inserting hop. \n", err.Error())
	}
	db.Close()
	dataReturned.Close()

}

//================== SELECT STATEMENTS ===================\\

//Selects all Syncboxes in the DB
func SelectAllSyncboxes() []string {
	var err error
	var syncboxList []string

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool \n" + err.Error())
	}
	defer db.Close()

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		fmt.Println("Ping error: ", err.Error())
	}

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

func SelectMtrReportByID(reportIds []string) []dataObjects.MtrReport {

	var err error
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool \n" + err.Error())
	}

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		fmt.Println("Error selecting mtr report. ", err.Error())
	}
	var dataReturned *sql.Rows
	var reportsReturned []dataObjects.MtrReport
	defer db.Close()
	for _, r := range reportIds {
		dataReturned, err = db.QueryContext(ctx, "sp_SelectMtrReportByID", r)
		if err != nil {
			fmt.Println("Error selecting mtr report. ", err.Error())
		}

		reportsReturned = parseSqlDataIntoReport(dataReturned)
	}

	//Troubleshooting
	// for _, r := range reportsReturned {
	// 	r.PrintReport()
	// }

	dataReturned.Close()

	return reportsReturned
}

func parseSqlDataIntoReport(sqlRowData *sql.Rows) []dataObjects.MtrReport {
	var reports []dataObjects.MtrReport
	var report dataObjects.MtrReport
	var reportID, syncboxID, dataCenter, hostName *string
	var startTime *time.Time
	var hopID, hopNumber, packetsSent *int
	var packetLoss, last, avg, best, worst, stdDev *float32
	for sqlRowData.Next() {
		if err := sqlRowData.Scan(&reportID, &syncboxID, &startTime, &dataCenter, &hopID, &hopNumber,
			&hostName, &packetLoss, &packetsSent, &last, &avg, &best, &worst, &stdDev); err != nil {
			panic(err.Error())
		} else {
			//fmt.Println(*reportID == report.ReportID)
			if *reportID == report.ReportID {
				hop := dataObjects.MtrHop{HopID: *hopID, HopNumber: *hopNumber, Hostname: *hostName,
					PacketLoss: *packetLoss, PacketsSent: *packetsSent, LastPing: *last, AveragePing: *avg,
					BestPing: *best, WorstPing: *worst, StdDev: *stdDev}
				report.Hops = append(report.Hops, hop)
				//fmt.Println(len(report.Hops))
			} else {
				//fmt.Println(len(report.Hops))
				reports = append(reports, report)
				report = dataObjects.MtrReport{}
				report = dataObjects.MtrReport{ReportID: *reportID, SyncboxID: *syncboxID, StartTime: *startTime, DataCenter: *dataCenter}
				hop := dataObjects.MtrHop{HopID: *hopID, HopNumber: *hopNumber, Hostname: *hostName,
					PacketLoss: *packetLoss, PacketsSent: *packetsSent, LastPing: *last, AveragePing: *avg,
					BestPing: *best, WorstPing: *worst, StdDev: *stdDev}
				report.Hops = append(report.Hops, hop)
			}
			//fmt.Println(*reportID + "|" + report.ReportID)

		}

		// for _, r := range reports {
		// 	fmt.Println(r)
		// }

	}
	reports = append(reports, report)

	//fmt.Println(reports)
	return reports
}

func CheckIfMtrReportExists(mtrReportID string) bool {
	var err error
	var mtrExists bool
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic("Error creating connection pool \n" + err.Error())
	}

	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		time.Sleep(50 * time.Millisecond)
		fmt.Println("Error checking mtr report. ", err.Error())
	}

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

func SelectSyncboxMtrReportsByDCAndTimeframe(syncbox string, startTime time.Time, endTime time.Time, datacenter string) []dataObjects.MtrReport {

	return make([]dataObjects.MtrReport, 0)
}
