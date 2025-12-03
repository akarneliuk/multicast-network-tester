/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Sender logic.
*/
package main

// Import
import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// Aux functions
func craftMessage(ctr uint64) ([]byte, error) {
	/* Helper message to produce message */

	// time in nanosecons since midnight UTC
	msgTimestamp := getTimestampFromMidnight()

	// Prepare message
	msg := MessageFormat{
		Timestamp: msgTimestamp,
		Num:       ctr,
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

	// Return
	return msgBin, nil
}

func sendIPv4MulitcastLoop(c chan MulticastGroup, conn *net.UDPConn, egressIf *net.Interface, dstAddr net.Addr, mg *MulticastGroup) {
	// Setup connection for random
	pc := ipv4.NewPacketConn(conn)

	// Set TTL and Egress Interfaces
	pc.SetMulticastTTL(CLI.Sender.TTL)
	pc.SetMulticastInterface(egressIf)

	// Create control channel
	if err := pc.SetControlMessage(ipv4.FlagDst, true); err != nil {
		logger.Fatalf("Cannot open the control channel to kernel. Error: %v\n", err)
	}

	// Message counter
	msgCounter := uint64(0)

	// Send packets
	for {
		msgBin, err := craftMessage(msgCounter)

		if err != nil {
			logger.Fatalf("Cannot encode message to send over wire. Error: %v\n", err)
		}

		// Send it
		n, err := pc.WriteTo(msgBin, nil, dstAddr)
		if err != nil {
			logger.Fatalf("Cannot send UDP message. Error: %v\n", err)
		}

		// Update with send bytes
		(*mg).Bytes = uint64(n)

		// Inform back about sent data
		c <- (*mg)

		// random sleep
		msgCounter++
		time.Sleep(time.Duration(rand.Uint64N(100)) * 10 * time.Millisecond)
	}
}

func sendIPv6MulitcastLoop(c chan MulticastGroup, conn *net.UDPConn, egressIf *net.Interface, dstAddr net.Addr, mg *MulticastGroup) {
	// Setup connection for random
	pc := ipv6.NewPacketConn(conn)

	// Set TTL and Egress Interfaces
	pc.SetMulticastHopLimit(CLI.Sender.TTL)
	pc.SetMulticastInterface(egressIf)

	// Create control channel
	if err := pc.SetControlMessage(ipv6.FlagDst, true); err != nil {
		logger.Fatalf("Cannot open the control channel to kernel. Error: %v\n", err)
	}

	// Message counter
	msgCounter := uint64(0)

	// Send packets
	for {
		msgBin, err := craftMessage(msgCounter)

		if err != nil {
			logger.Fatalf("Cannot encode message to send over wire. Error: %v\n", err)
		}

		// Send it
		n, err := pc.WriteTo(msgBin, nil, dstAddr)
		if err != nil {
			logger.Fatalf("Cannot send UDP message. Error: %v\n", err)
		}

		// Update with send bytes
		(*mg).Bytes = uint64(n)

		// Inform back about sent data
		c <- (*mg)

		// random sleep
		msgCounter++
		time.Sleep(time.Duration(rand.Uint64N(100)) * 10 * time.Millisecond)
	}
}

// Core functions
func (mg MulticastGroup) sendPackets(c chan MulticastGroup) {
	/* Sending Multicast traffic */
	// Get egress interface
	egressIf, err := net.InterfaceByName(mg.Iface)
	if err != nil {
		logger.Fatalf("Cannot get egress Interface: %v", err)
	}

	// Get egress interface IP addresses
	egressIfAddresses, err := egressIf.Addrs()
	if err != nil {
		logger.Fatalf("Cannot get egress Interface IP addresses: %v", err)
	}

	var srcIP net.IP
	var localUdpAddr, dstAddr *net.UDPAddr

	for _, addr := range egressIfAddresses {
		srcIP = net.ParseIP(strings.Split(addr.String(), "/")[0])
		if srcIP.To4() != nil && mg.GrpAddress.To4() != nil {
			// IPv4
			localUdpAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:0", srcIP.String()))
			if err != nil {
				logger.Fatalf("Cannot get free UDP port for IPv4: %v", err)
			}

			break
		} else if srcIP.To4() == nil && mg.GrpAddress.To4() == nil {
			// IPv4
			localUdpAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("[%s]:0", srcIP.String()))
			if err != nil {
				logger.Fatalf("Cannot get free UDP port for IPv6: %v", err)
			}
			break
		}
	}
	if srcIP == nil {
		logger.Fatalf("Cannot determinte egress interface IP addess")
	}

	// Start listener
	conn, err := net.ListenUDP("udp", localUdpAddr)
	if err != nil {
		logger.Fatalf("Cannot create UDP socket: %s", err)
	}
	defer conn.Close()

	// Start sending packets
	if mg.GrpAddress.To4() == nil {
		// IPv6

		// Get destination addr
		dstAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("[%s]:%d", mg.GrpAddress.String(), mg.Port))
		if err != nil {
			logger.Fatalf("Cannot resolve destination IPv4/port: %v\n", err)
		}

		sendIPv6MulitcastLoop(c, conn, egressIf, dstAddr, &mg)

	} else {
		// IPV4

		// Get destination addr
		dstAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", mg.GrpAddress.String(), mg.Port))
		if err != nil {
			logger.Fatalf("Cannot resolve destination IPv6/port: %v\n", err)
		}

		sendIPv4MulitcastLoop(c, conn, egressIf, dstAddr, &mg)
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
