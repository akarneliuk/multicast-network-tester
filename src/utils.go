/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Shared functions
*/
package main

// Import
import "time"

// Functions
func getTimestampFromMidnight() int64 {
	/* Helper function to get timestmap in nanoseconds from the midnight. */
	tN := time.Now().UTC()
	tM := time.Date(tN.Year(), tN.Month(), tN.Day(), 0, 0, 0, 0, time.UTC)
	return tN.Sub(tM).Nanoseconds()
}
