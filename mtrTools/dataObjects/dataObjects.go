package dataObjects

import (
	"fmt"
	"strings"
	"time"
)

type MtrHop struct {
	HopID       int     `json:"HopID"`
	ReportID    int     `json:"ReportID"`
	HopNumber   int     `json:"HopNumber"`
	Hostname    string  `json:"Hostname"`
	PacketLoss  float32 `json:"PacketLoss"`
	PacketsSent int     `json:"PacketsSent"`
	LastPing    float32 `json:"LastPing"`
	AveragePing float32 `json:"AveragePing"`
	BestPing    float32 `json:"BestPing"`
	WorstPing   float32 `json:"WorstPing"`
	StdDev      float32 `json:"StdDev"`
}

type MtrReport struct {
	ReportID    int       `json:"ReportID"`
	SyncboxID   string    `json:"SyncboxID"`
	StartTime   time.Time `json:"StartTime"`
	DataCenter  string    `json:"DataCenter"`
	Success     bool      `json:"Success"`
	Hops        []MtrHop  `json:"Hops"`
	LogFilename string
}

// Prints out an Mtr Report with properly aligned fields
func (rpt MtrReport) PrintReport() string {
	reportString := ""
	reportString += fmt.Sprint(rpt.ReportID) + "\n" +
		rpt.SyncboxID + "\n" +
		fmt.Sprint(rpt.StartTime.Format(time.ANSIC)) + "\n" +
		"Data Center: " + strings.ToUpper(rpt.DataCenter) + "\n"
	longestHostname := ""
	for _, h := range rpt.Hops {
		if len(h.Hostname) > len(longestHostname) {
			longestHostname = h.Hostname
		}
	}
	hostHeader := ""
	for i := 0; i <= len(longestHostname); i++ {
		hostHeader = hostHeader + " "
		if i == ((len(longestHostname) / 2) - 2) {
			hostHeader = hostHeader + "Host"
			i = i + 4
		}
	}
	reportString += "Hop|" + hostHeader + "\t|Loss%\t|Sent\t|Last\t|Avg\t|Best\t|Worst\t|Std\n"

	for _, h := range rpt.Hops {
		hopnum := fmt.Sprint(h.HopNumber)
		if h.HopNumber < 10 {
			hopnum = hopnum + "  "
		} else {
			hopnum = hopnum + " "
		}

		hostname := h.Hostname
		for i := 0; i <= (len(longestHostname) - len(h.Hostname)); i++ {
			hostname = hostname + " "
		}
		packetLoss := fmt.Sprint(h.PacketLoss)
		if len(packetLoss) < 5 {
			packetLoss = packetLoss + "\t"
		}
		reportString += hopnum + "|" +
			hostname + "\t|" +
			packetLoss + "|" +
			fmt.Sprint(h.PacketsSent) + "\t|" +
			fmt.Sprint(h.LastPing) + "\t|" +
			fmt.Sprint(h.AveragePing) + "\t|" +
			fmt.Sprint(h.BestPing) + "\t|" +
			fmt.Sprint(h.WorstPing) + "\t|" +
			fmt.Sprint(h.StdDev) + "\n"
	}

	return reportString
}
