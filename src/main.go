/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Main business logic.
*/
package main

import (
	"github.com/alecthomas/kong"
)

// Function
func main() {
	// Get CLI arguments
	kong.Parse(&CLI)

	// Ensure that either sender or receiver arguments are specified
	switch {
	case CLI.Receiver.ConfigFile != "":
		logger.Println("Starting multicast receiver...")
		startReceiver()

	case len(CLI.Sender.MulticastGroups) > 0:
		logger.Println("Starting multicast sender...")
		startSender()

	default:
		logger.Fatalf("No configuration was provider for sender or receiver. Run 'multicast-tester -h' for further details.")
	}
}
