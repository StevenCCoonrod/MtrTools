package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

// Verifies that the syncbox arguments exist in the Syncbox list,
// and returns a list of syncboxes based on each argument provided
func processSyncboxArgs(syncboxArgs []string) []string {
	var syncboxes []string
	for _, s := range syncboxArgs {
		s = strings.ToLower(s)
		if !strings.Contains(s, "-2309") {
			s = s + "-2309"
			var autoBox []string
			if string(byte(s[0])) == "k" {
				if slices.Contains(_SyncboxList, strings.ToUpper(s)) {
					autoBox = append(autoBox, s)
				}
				l := strings.Replace(s, "k", "l", 1)
				if slices.Contains(_SyncboxList, strings.ToUpper(l)) {
					autoBox = append(autoBox, l)
				}
				m := strings.Replace(s, "k", "m", 1)
				if slices.Contains(_SyncboxList, strings.ToUpper(m)) {
					autoBox = append(autoBox, m)
				}
				n := strings.Replace(s, "k", "n", 1)
				if slices.Contains(_SyncboxList, strings.ToUpper(n)) {
					autoBox = append(autoBox, n)
				}

			} else if string(byte(s[0])) == "w" {
				if slices.Contains(_SyncboxList, strings.ToUpper(s)) {
					autoBox = append(autoBox, s)
				}
				x := strings.Replace(s, "w", "x", 1)
				if slices.Contains(_SyncboxList, strings.ToUpper(x)) {
					autoBox = append(autoBox, x)
				}
				y := strings.Replace(s, "w", "y", 1)
				if slices.Contains(_SyncboxList, strings.ToUpper(y)) {
					autoBox = append(autoBox, y)
				}
				z := strings.Replace(s, "w", "z", 1)
				if slices.Contains(_SyncboxList, strings.ToUpper(z)) {
					autoBox = append(autoBox, z)
				}
			} else {
				if slices.Contains(_SyncboxList, strings.ToUpper(s)) {
					autoBox = append(autoBox, s)
				}
			}
			syncboxes = append(syncboxes, autoBox...)
		} else {
			if slices.Contains(_SyncboxList, strings.ToUpper(s)) {
				syncboxes = append(syncboxes, s)
			}
		}

	}
	return syncboxes
}

// Displays program details to console
func programDisplay() {
	fmt.Print("\n===============================================")
	fmt.Print(" Mtr Tools ")
	fmt.Println("===============================================")
	fmt.Println("Flags:")
	fmt.Println("\t-a\tRuns a sweep of ALL Syncboxes")
	fmt.Println("\t-start\tSpecifies a target search start time. Eg. 5h30m = 5 hours and 30 minutes ago")
	fmt.Println("\t-end\tSpecifies a target search end time. Eg. 0m = now, 0 minutes ago")
	fmt.Println("\t-p\tPrints results to the command line")
	fmt.Println("\t-pf\tPrint the results to a text file")
	fmt.Println("Syncboxes:", len(_SyncboxList))
	for i, s := range _SyncboxList {
		if i%7 == 0 {
			fmt.Print("\n" + s)
		} else {
			fmt.Print("\t" + s)
		}
	}
}

// Helper function to validate a timeframe
func validateTimeframe(startTime time.Duration, endTime time.Duration) bool {
	// Check for valid start and end times
	validTimes := false

	switch {
	// Check if start time is 5 minutes in the past
	case !time.Now().Add(-startTime).Before(time.Now().Add(-time.Minute * 4)):
		validTimes = false
		fmt.Println("Start time must be 5 minutes or more ago.")
	// Check if end time is after time.Now() time.Now()
	case !(time.Now().Add(-endTime).Before(time.Now()) || time.Now().Add(-endTime).Equal(time.Now())):
		validTimes = false
		fmt.Println("End time cannot be in the future")
	// Check if end time is before start time
	case !time.Now().Add(-endTime).After(time.Now().Add(-startTime)):
		validTimes = false
		fmt.Println("End time must be after start time.")
	// If none of these cases arise, times are valid
	default:
		validTimes = true
	}

	return validTimes
}

// Sets the values provided for the flags accepted by the program
func setFlags() (time.Duration, time.Duration, string, string) {
	var all bool
	flag.BoolVar(&all, "a", false, "Target ALL syncboxes")
	defaultTime := time.Since(time.Now())
	var startTime time.Duration
	flag.DurationVar(&startTime, "start", defaultTime, "Search timeframe start time")
	var endTime time.Duration
	flag.DurationVar(&endTime, "end", defaultTime, "Search timeframe end time")
	var printResult bool
	flag.BoolVar(&printResult, "p", false, "Print search results to command-line")
	var filterByDataCenter string
	flag.StringVar(&filterByDataCenter, "dc", "", "Filter search results by data center")
	var printToFile bool
	flag.BoolVar(&printToFile, "pf", false, "Print results to a text file")
	var hostname string
	flag.StringVar(&hostname, "host", "", "View reports involving the host name provided")

	flag.Parse()
	return startTime, endTime, filterByDataCenter, hostname
}

// Verifies if a flag has been provided
func IsFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
