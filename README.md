# Multicast Network Tester

This is an application to test multicast network. Written in pure Go.

Key functionality:
- Supports both IPv4 and IPv6
- Support multiple channels (IPv4 only, IPv6 only, or mixture of both)
- Exports metrics via Prometheus:
  - Packets received
  - Packets out of order received
  - Bytes received
  - Latency between sender and receiver

## Build
```shell
$ mkdir bin
$ go build -C src/ -o ../bin/. -v
```

## Usage
### Sender
Pass the same config file to the sender:
```shell
$ ./bin/multicast-tester sender --ttl=64 \
  [ff03::1]:5000@wlp0s20f3 \
  239.0.0.1:5000@wlp0s20f3 \
  [ff03::2]:5001@wlp0s20f3 \
  239.0.1.1:5002@wlp0s20f3
```

### Receiver
Create config file:
```shell
$ tee bin/config.yaml << __EOF__
---
prometheus:
  enabled: true
  port: 9876
multicast_channels:
  - name: "Channel 1 - A feed"
    interface: "wlp0s20f3"
    group_address: "239.0.0.1"
    port: 5000
  - name: "Channel 1 - B feed"
    interface: "wlp0s20f3"
    group_address: "239.0.1.1"
    port: 5002
  - name: "Channel 2 - A feed"
    interface: "wlp0s20f3"
    group_address: "ff03::1"
    port: 5000
  - name: "Channel 2 - B feed"
    interface: "wlp0s20f3"
    group_address: "ff03::2"
    port: 5001
__EOF__
```
Specify the interface, where you want to receive multicast packets, and the multicast groups you want to listen to.

Start the receiver:
```shell
$ ./bin/multicast-tester receiver ./bin/config.yaml
```

#### Stats
Check the counters of received packets:
```bash
$ curl http://localhost:9876/metrics
# HELP multicast_bytes_received Amount of bytes received in multicast packets.
# TYPE multicast_bytes_received counter
multicast_bytes_received{grp_address="239.0.0.1",port="5000",src_address="192.168.0.2"} 3576
multicast_bytes_received{grp_address="239.0.1.1",port="5002",src_address="192.168.0.2"} 168
multicast_bytes_received{grp_address="ff03::1",port="5000",src_address="2a04:****::ab83"} 3384
multicast_bytes_received{grp_address="ff03::2",port="5001",src_address="2a04:****::ab83"} 3648
# HELP multicast_latency_nanoseconds Latency in nanoseconds between packet created by sender and parsed by receiver.
# TYPE multicast_latency_nanoseconds gauge
multicast_latency_nanoseconds{grp_address="239.0.0.1",port="5000",src_address="192.168.0.2"} 96607
multicast_latency_nanoseconds{grp_address="239.0.1.1",port="5002",src_address="192.168.0.2"} 96679
multicast_latency_nanoseconds{grp_address="ff03::1",port="5000",src_address="2a04::****:::5576"} 103602
multicast_latency_nanoseconds{grp_address="ff03::2",port="5001",src_address="2a04::****:::5576"} 190396
# HELP multicast_packets_out_of_order Number of multicast packets received out of order since start.
# TYPE multicast_packets_out_of_order counter
multicast_packets_out_of_order{grp_address="239.0.0.1",port="5000",src_address="192.168.0.2"} 0
multicast_packets_out_of_order{grp_address="239.0.1.1",port="5002",src_address="192.168.0.2"} 0
multicast_packets_out_of_order{grp_address="ff03::1",port="5000",src_address="2a04:****::ab83"} 0
multicast_packets_out_of_order{grp_address="ff03::2",port="5001",src_address="2a04:****::ab83"} 0
# HELP multicast_packets_received Number of multicast packets received since start.
# TYPE multicast_packets_received counter
multicast_packets_received{grp_address="239.0.0.1",port="5000",src_address="192.168.0.2"} 149
multicast_packets_received{grp_address="239.0.1.1",port="5002",src_address="192.168.0.2"} 7
multicast_packets_received{grp_address="ff03::1",port="5000",src_address="2a04:****::ab83"} 141
multicast_packets_received{grp_address="ff03::2",port="5001",src_address="2a04:****::ab83"} 152
# HELP promhttp_metric_handler_errors_total Total number of internal errors encountered by the promhttp metric handler.
# TYPE promhttp_metric_handler_errors_total counter
promhttp_metric_handler_errors_total{cause="encoding"} 0
promhttp_metric_handler_errors_total{cause="gathering"} 0
```

## To-do
1. Telemetry on sender
2. Sending telemetry from receiver to sender
3. Variable message size up to max MTU on interface
4. Sending telemetry to OTEL
