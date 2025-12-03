/*
Multicast Network Tester
(c) 2025, Anton Karneliuk
Receiver logic.
*/
package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.yaml.in/yaml/v2"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// Aux functions
func readConfig(filename string) (McastReceiverConfig, error) {
	/* Helper function to load the configuration of receiver */
	// Result
	result := McastReceiverConfig{}

	// Read input file
	bs, err := os.ReadFile(filename)
	if err != nil {
		return result, err
	}

	// Parse YAML
	err = yaml.Unmarshal(bs, &result)
	if err != nil {
		return result, err
	}

	// Return result
	return result, nil
}

func (mg MulticastGroup) startMulticastListener(c chan MulticastGroupMetrics) {
	/* Main function to receive multicast data. */
	// Close channel once done
	defer func() { close(c) }()

	// Get interface
	iface, err := net.InterfaceByName(mg.Iface)
	if err != nil {
		logger.Fatalf("Cannot find the interface %v. Error: %v\n", mg.Iface, err)
	}

	// Create listening socket
	listenString := new(string)
	if mg.GrpAddress.To4() == nil {
		*listenString = fmt.Sprintf("[%v]:%d", mg.GrpAddress, mg.Port)
	} else {
		*listenString = fmt.Sprintf("%v:%d", mg.GrpAddress, mg.Port)
	}

	conn, err := net.ListenPacket("udp", *listenString)
	if err != nil {
		logger.Fatalf("Cannot listen to %v:%d. Error: %v\n", mg.GrpAddress.String(), mg.Port, err)
	}
	defer conn.Close()

	// Initialise metrics
	previousPacketNumber := uint64(0)
	outOfOrderPacket := uint64(0)
	rxLatency := int64(0)

	///////////////
	// IPv6 part //
	///////////////
	if mg.GrpAddress.To4() == nil {
		pc := ipv6.NewPacketConn(conn)

		localMulticastAddress := net.UDPAddr{IP: mg.GrpAddress, Port: int(mg.Port)}
		if err := pc.JoinGroup(iface, &localMulticastAddress); err != nil {
			logger.Fatalf("Cannot join multicast group %v on interface %v. Error: %v\n", mg.GrpAddress, iface, err)
		}
		defer pc.LeaveGroup(iface, &localMulticastAddress)

		// Control channel
		if err := pc.SetControlMessage(ipv6.FlagDst, true); err != nil {
			logger.Fatalf("Cannot open the control channel to kernel. Error: %v\n", err)
		}

		// Notification
		logger.Printf("Ready to receive packets for group %v/%d on %v\n", mg.GrpAddress.String(), mg.Port, iface.Name)

		// Reading packets
		packet := make([]byte, 1500)
		for {
			n, cm, src, err := pc.ReadFrom(packet)
			if err != nil {
				logger.Fatalf("Cannot read received packet. Error: %v\n", err)
			}

			// Decode packet
			msg := MessageFormat{
				Timestamp: int64(binary.BigEndian.Uint64(packet)),
				Num:       binary.BigEndian.Uint64(packet[8:]),
				Kind:      binary.BigEndian.Uint16(packet[16:]),
			}

			// Calculate rxLatency
			rxLatency = getTimestampFromMidnight() - msg.Timestamp

			// Check the packet is out of order
			if previousPacketNumber == 0 {
				previousPacketNumber = msg.Num
				outOfOrderPacket = 0
			} else {
				if msg.Num-previousPacketNumber != uint64(1) {
					outOfOrderPacket = 1
				} else {
					outOfOrderPacket = 0
				}
				previousPacketNumber = msg.Num
			}

			// Print decoded packet for debug purpose
			if CLI.Receiver.IsDebug {
				logger.Printf("%+v\n", msg)
			}

			// Check if received packet is multicast
			if cm.Dst.IsMulticast() {
				// Check that packet is matching group
				if cm.Dst.Equal(mg.GrpAddress) {
					received := MulticastGroupMetrics{
						SrcAddress: net.ParseIP(strings.Split(strings.Split(src.String(), "]:")[0], "[")[1]),
						GrpAddress: mg.GrpAddress,
						Port:       mg.Port,
						Bytes:      uint64(n),
						OutOfOrder: outOfOrderPacket,
						RxLatency:  rxLatency,
					}

					// Send info to channel
					c <- received
				} else if CLI.Receiver.IsDebug {
					logger.Printf("Received packet for unknown channel (%v:%v), ignoring it.", src.String(), mg.GrpAddress.String())
				}
			}
		}
		///////////////
		// IPv4 part //
		///////////////
	} else {
		// Join multicast group
		pc := ipv4.NewPacketConn(conn)

		localMulticastAddress := net.UDPAddr{IP: mg.GrpAddress, Port: int(mg.Port)}
		if err := pc.JoinGroup(iface, &localMulticastAddress); err != nil {
			logger.Fatalf("Cannot join multicast group %v on interface %v. Error: %v\n", mg.GrpAddress, iface, err)
		}
		defer pc.LeaveGroup(iface, &localMulticastAddress)

		// Control channel
		if err := pc.SetControlMessage(ipv4.FlagDst, true); err != nil {
			logger.Fatalf("Cannot open the control channel to kernel. Error: %v\n", err)
		}

		// Notification
		logger.Printf("Ready to receive packets for group %v/%d on %v\n", mg.GrpAddress.String(), mg.Port, iface.Name)

		// Reading packets
		packet := make([]byte, 1500)
		for {
			n, cm, src, err := pc.ReadFrom(packet)
			if err != nil {
				logger.Fatalf("Cannot read received packet. Error: %v\n", err)
			}

			// Decode packet
			msg := MessageFormat{
				Timestamp: int64(binary.BigEndian.Uint64(packet)),
				Num:       binary.BigEndian.Uint64(packet[8:]),
				Kind:      binary.BigEndian.Uint16(packet[16:]),
			}

			// Calculate rxLatency
			rxLatency = getTimestampFromMidnight() - msg.Timestamp

			// Check the packet is out of order
			if previousPacketNumber == 0 {
				previousPacketNumber = msg.Num
				outOfOrderPacket = 0
			} else {
				if msg.Num-previousPacketNumber != uint64(1) {
					outOfOrderPacket = 1
				} else {
					outOfOrderPacket = 0
				}
				previousPacketNumber = msg.Num
			}

			// Print decoded packet for debug purpose
			if CLI.Receiver.IsDebug {
				logger.Printf("%+v\n", msg)
			}

			// Check if received packet is multicast
			if cm.Dst.IsMulticast() {
				// Check that packet is matching group
				if cm.Dst.Equal(mg.GrpAddress) {
					received := MulticastGroupMetrics{
						SrcAddress: net.ParseIP(strings.Split(src.String(), ":")[0]),
						GrpAddress: mg.GrpAddress,
						Port:       mg.Port,
						Bytes:      uint64(n),
						OutOfOrder: outOfOrderPacket,
						RxLatency:  rxLatency,
					}

					// Send info to channel
					c <- received
				} else if CLI.Receiver.IsDebug {
					logger.Printf("Received packet for unknown channel (%v:%v), ignoring it.", src.String(), mg.GrpAddress.String())
				}
			}
		}
	}
}

func NewPromMetrics(reg prometheus.Registerer) *PrometheusMetrics {
	/* Create prometheus metrics for exporting */
	// Define labeles
	appLabels := []string{"src_address", "grp_address", "port"}

	// Create mertics
	m := &PrometheusMetrics{
		multicastPacketsReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "multicast_packets_received",
				Help: "Number of multicast packets received since start.",
			},
			appLabels,
		),
		multicastPacketsOutOfOrder: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "multicast_packets_out_of_order",
				Help: "Number of multicast packets received out of order since start.",
			},
			appLabels,
		),
		multicastBytesReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "multicast_bytes_received",
				Help: "Amount of bytes received in multicast packets.",
			},
			appLabels,
		),
		multicastLatencyNanoseconds: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "multicast_latency_nanoseconds",
				Help: "Latency in nanoseconds between packet created by sender and parsed by receiver.",
			},
			appLabels,
		),
	}

	// Register metrics at registry
	reg.MustRegister(m.multicastPacketsReceived)
	reg.MustRegister(m.multicastPacketsOutOfOrder)
	reg.MustRegister(m.multicastBytesReceived)
	reg.MustRegister(m.multicastLatencyNanoseconds)

	// Return result
	return m
}

func startPrometheusServer(reg *prometheus.Registry, ac McastReceiverConfig) {
	/* Helper process to handle Prometheus requests */
	// Expose handler
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", ac.PromConfig.Port), nil))
}

func startReceiver() {
	/* Main business logic for Reciver */

	// Load configuration
	appConfig, err := readConfig(CLI.Receiver.ConfigFile)
	if err != nil {
		logger.Fatalf("Failed to load the configuration. Error: %v\n", err)
	}

	// Response channel
	mcc := make(chan MulticastGroupMetrics)

	// Start multicast listeners
	for _, mgr := range appConfig.MulticastGroups {
		go mgr.startMulticastListener(mcc)
	}

	// Prometheus part
	reg := prometheus.NewRegistry()
	promMetr := NewPromMetrics(reg)

	// Start Prometheus server
	if appConfig.PromConfig.Enabled {
		go startPrometheusServer(reg, appConfig)
	}

	// Recive data (prod)
	for r := range mcc {
		if CLI.Receiver.IsDebug {
			fmt.Printf("%+v\n", r)
		}

		// Update Prometheus metrics
		if appConfig.PromConfig.Enabled {
			promMetr.multicastPacketsReceived.With(prometheus.Labels{"src_address": r.SrcAddress.String(), "grp_address": r.GrpAddress.String(), "port": fmt.Sprint(r.Port)}).Inc()
			promMetr.multicastPacketsOutOfOrder.With(prometheus.Labels{"src_address": r.SrcAddress.String(), "grp_address": r.GrpAddress.String(), "port": fmt.Sprint(r.Port)}).Add(float64(r.OutOfOrder))
			promMetr.multicastBytesReceived.With(prometheus.Labels{"src_address": r.SrcAddress.String(), "grp_address": r.GrpAddress.String(), "port": fmt.Sprint(r.Port)}).Add(float64(r.Bytes))
			promMetr.multicastLatencyNanoseconds.With(prometheus.Labels{"src_address": r.SrcAddress.String(), "grp_address": r.GrpAddress.String(), "port": fmt.Sprint(r.Port)}).Set(float64(r.RxLatency))
		}
	}

	// Result
	fmt.Println("Job done")
}
