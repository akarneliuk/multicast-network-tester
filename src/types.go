/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Shared data types.
*/
package main

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/prometheus/client_golang/prometheus"
)

// Types
type CliArg struct {
	/* CLI configuration for multicast tester.
	Shared - used both in Sender and Receiver. */

	Sender struct {
		MulticastGroups MulticastGroups `arg help:"Multicast groups in format 'ip_address:port@interface'." type:"string"`
		IsDebug         bool            `short:"d" help:"Enable debug mode"`
	} `cmd help:"Multicast sender commands."`

	Receiver struct {
		ConfigFile string `arg help:"Configuration file with multicast subscription." type:"string"`
		IsDebug    bool   `short:"d" help:"Enable debug mode"`
	} `cmd help:"Multicast receiver commands."`
}
type MulticastGroup struct {
	/* Shared - used both in Sender and Receiver. */
	Name       string `yaml:"name"`
	Iface      string `yaml:"interface"`
	SrcAddress net.IP `yaml:"source_address"`
	GrpAddress net.IP `yaml:"group_address"`
	Port       uint16 `yaml:"port"`
	Bytes      uint64 `yaml:"bytes"`
}
type MulticastGroups []MulticastGroup
type MessageFormat struct {
	/* Shared - used both in Sender and Receiver. */
	Timestamp int64
	Num       uint64
	Kind      uint16 // 0 - test, 1 - operation
}
type McastReceiverConfig struct {
	/* Receiver only. */
	PromConfig      `yaml:"prometheus"`
	MulticastGroups `yaml:"multicast_channels"`
}
type PromConfig struct {
	/* Receiver only. */
	Enabled bool   `yaml:"enabled"`
	Port    uint16 `yaml:"port"`
}
type PrometheusMetrics struct {
	/* Receiver only. */
	multicastPacketsReceived *prometheus.CounterVec
	multicastBytesReceived   *prometheus.CounterVec
}

// Receivers for parsing
func (mgs *MulticastGroups) Decode(ctx *kong.DecodeContext) error {
	/* Custom parser for Kong for Multicast Groups */

	// Loop Through argument strings
	argsNo := (*ctx).Scan.Len()
	for i := 0; i < argsNo; i++ {
		v := (*ctx).Scan.Pop()
		ifaceAndAdressPort := strings.Split(v.String(), "@")
		if len(ifaceAndAdressPort) != 2 {
			return fmt.Errorf("'%v' doesn't contain IP address/port and/or interface", ifaceAndAdressPort)
		}

		// Parse IP and Port
		var mcastAddr net.IP
		var mcastPort uint64
		var mcastAddrPort []string

		// Check if IPv6 or IPv4
		reIPv6 := regexp.MustCompile(`^\[[a-fA-F0-9:]+\]:\d+$`)
		reIPv4 := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+`)
		switch {
		case reIPv6.MatchString(ifaceAndAdressPort[0]):
			mcastAddrPort = strings.Split(ifaceAndAdressPort[0], "]:")
			mcastAddrPort[0] = strings.Trim(mcastAddrPort[0], "[")

		case reIPv4.MatchString(ifaceAndAdressPort[0]):
			mcastAddrPort = strings.Split(ifaceAndAdressPort[0], ":")

		default:
			return fmt.Errorf("'%v' is of unknown format. Shall be 239.2.3.4:123 for IPv4 or [ff03::123]:123 for IPv6", ifaceAndAdressPort[0])
		}

		// Check that both address and port exist
		if len(mcastAddrPort) != 2 {
			return fmt.Errorf("'%v' doesn't contain IP address and/or port", ifaceAndAdressPort[0])
		}

		mcastAddr = net.ParseIP(mcastAddrPort[0])
		mcastPort, err := strconv.ParseUint(mcastAddrPort[1], 10, 64)
		if err != nil {
			return err
		}

		*mgs = append(*mgs, MulticastGroup{Iface: ifaceAndAdressPort[1], GrpAddress: mcastAddr, Port: uint16(mcastPort)})
	}

	// Assign using reflection
	ctx.Value.Apply(reflect.ValueOf(*mgs))

	// Return error
	return nil
}
