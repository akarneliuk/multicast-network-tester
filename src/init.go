/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Initiation
*/
package main

import (
	"log"
	"os"
)

// Vars
var (
	logger *log.Logger
	CLI    CliArg
)

// Init
func init() {
	/* Initialize app */
	// Logger
	logger = log.New(os.Stdout, "MulticastSender: ", log.Lmicroseconds|log.Ldate|log.LUTC|log.Lmsgprefix)

}
