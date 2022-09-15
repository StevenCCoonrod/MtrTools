package main

import (
	"fmt"
	"mtrTools/dataObjects"
	"mtrTools/sshDataAccess"
	"sync"
	"time"
)

// Retrieves ALL MTR logs for ALL syncboxes in the SyncboxList
func fullMtrRetrievalCycle() {
	fmt.Println("============ Initiating Full MTR Sweep ============")
	fmt.Println("\nFull Sweep Initiated At", time.Now().UTC().Format(time.ANSIC))
	batchCount := 15
	batches := make([][]string, len(_SyncboxList)/batchCount+1)
	position := 0
	for i, s := range _SyncboxList {
		if i != 0 && i%batchCount == 0 {
			position += 1
		}
		batches[position] = append(batches[position], s)
	}

	var wg sync.WaitGroup

	ch := make(chan []string)

	// for i, b := range batches {
	// 	wg.Add(1)
	// 	fmt.Println("Working on Batch", i, ":", b)
	// 	go getBatchMtrData(&wg, b, time.Since(time.Now().UTC().AddDate(0, 0, -1)), time.Duration(0))
	// 	wg.Wait()
	// 	fmt.Println("Batch", b, "completed")
	// }

	go func() {
		for i, b := range batches {

			ch <- b

			if i%5 == 0 {
				time.Sleep(time.Second * 5)
			}

		}
		close(ch)

	}()

	for batch := range ch {
		wg.Add(1)
		fmt.Println("Working on Batch:", batch)
		go getBatchMtrData(&wg, batch, time.Since(time.Now().UTC().AddDate(0, 0, -1)), time.Duration(0))
		//wg.Wait()
		fmt.Println("Batch", batch, "completed")
	}
	wg.Wait()
	fmt.Println("============ MTR Sweep Completed ============")
}

func startBatchChannel() {

}

func getBatchMtrData(wg *sync.WaitGroup, syncboxes []string, startTime time.Duration, endTime time.Duration) []dataObjects.MtrReport {

	var batchReports []dataObjects.MtrReport
	//var syncboxStatus string
	//Get datetimes based on provided durations
	start := time.Now().UTC().Add(-startTime)
	end := time.Now().UTC().Add(-endTime)

	//Print Console Header
	if !IsFlagPassed("a") {
		fmt.Println("Start Time:\t" + fmt.Sprint(start.Format(time.ANSIC)) +
			"\nEnd Time:\t" + fmt.Sprint(end.Format(time.ANSIC)))

	}

	//For each syncbox provided, Check SSH, Insert any new reports, and return all reports found in DB

	batchReports = sshDataAccess.GetBatchMtrData_SpecificTimeframe(syncboxes, start, end)
	if batchReports == nil {

	}
	//Check SSH
	//batch, syncboxStatus = sshDataAccess.GetMtrData_SpecificTimeframe(syncbox, start, end)
	//Insert any new reports into the DB
	fmt.Println("Inserting into DB for:", syncboxes)
	insertMtrReportsIntoDB(batchReports)
	wg.Done()
	return batchReports
}
