package dataObjects

import (
	"fmt"
	"strings"
	"time"
)

type MtrHop struct {
	HopID       int
	ReportID    string
	HopNumber   int
	Hostname    string
	PacketLoss  float32
	PacketsSent int
	LastPing    float32
	AveragePing float32
	BestPing    float32
	WorstPing   float32
	StdDev      float32
}

type MtrReport struct {
	ReportID   string
	SyncboxID  string
	StartTime  time.Time
	DataCenter string
	Hops       []MtrHop
}

// Prints out an Mtr Report with properly aligned fields
func (rpt MtrReport) PrintReport() string {
	reportString := ""
	reportString += rpt.ReportID + "\n" +
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
