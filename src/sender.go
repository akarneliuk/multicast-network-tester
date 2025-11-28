/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Sender logic.
*/
package main

// Import
import (
	"encoding/binary"
	"log"
	"math/rand/v2"
	"net"
	"time"
	"unsafe"
)

// Core functions
func (mg MulticastGroup) sendPackets(c chan MulticastGroup) {
	/* Sending Multicast traffic */
	// Setup connection
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: mg.GrpAddress, Port: int(mg.Port)})
	if err != nil {
		logger.Fatalf("Cannot connect to UDP network. Error: %v\n", conn)
	}
	defer conn.Close()

	// Message counter
	msgCounter := uint64(0)

	// Send packets
	for {
		// time in nanosecons since midnight UTC
		tN := time.Now().UTC()
		tM := time.Date(tN.Year(), tN.Month(), tN.Day(), 0, 0, 0, 0, time.UTC)
		msgTimestamp := tN.Sub(tM).Nanoseconds()

		// Prepare message
		msg := MessageFormat{
			Timestamp: msgTimestamp,
			Num:       msgCounter,
			Kind:      0,
		}
		if CLI.Sender.IsDebug {
			logger.Printf("%+v\n", msg)
		}

		// Encode it to bytes
		msgBin := make([]byte, unsafe.Sizeof(MessageFormat{}))
		binary.BigEndian.PutUint64(msgBin, uint64(msg.Timestamp))
		binary.BigEndian.PutUint64(msgBin[8:], msg.Num)
		binary.BigEndian.PutUint16(msgBin[16:], msg.Kind)

		if err != nil {
			log.Fatalf("Cannot encode message to send over wire. Error: %v\n", err)
		}

		// Send it
		n, err := conn.Write(msgBin)
		if err != nil {
			logger.Fatalf("Cannot send UDP message. Error: %v\n", err)
		}

		// Update with send bytes
		mg.Bytes = uint64(n)

		// Inform back about sent data
		c <- mg

		// random sleep
		msgCounter++
		time.Sleep(time.Duration(rand.Uint64N(100)) * 10 * time.Millisecond)
	}

}

func startSender() {
	/* Main business logic for sender */

	// Create channel
	c := make(chan MulticastGroup)

	// Start sending packets
	for _, mg := range CLI.Sender.MulticastGroups {
		go mg.sendPackets(c)
	}

	// Receive notification form sending goroutine
	for r := range c {
		if CLI.Sender.IsDebug {
			logger.Printf("Sent %d bytes to %v:%d on %v\n", r.Bytes, r.GrpAddress, r.Port, r.Iface)
		}
	}
}
