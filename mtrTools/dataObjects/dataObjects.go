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

func (rpt MtrReport) PrintReport() {
	fmt.Println(rpt.ReportID)
	fmt.Println(rpt.SyncboxID)
	fmt.Println(rpt.StartTime)
	fmt.Println("Data Center: " + strings.ToUpper(rpt.DataCenter))
	for _, h := range rpt.Hops {
		fmt.Println(h.HopNumber, "\t", h.Hostname, "\t", h.PacketLoss, "\t", h.PacketsSent, "\t",
			h.LastPing, "\t", h.AveragePing, "\t", h.BestPing, "\t", h.WorstPing, "\t", h.StdDev)
	}
	fmt.Println()
}
