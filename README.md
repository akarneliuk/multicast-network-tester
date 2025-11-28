# Multicast Network Tester

This is an application to test multicast network. Written in pure Go.

Key functionality:
- Supports both IPv4 and IPv6
- Support multiple channels (IPv4 only, IPv6 only, or mixture of both)
- Exports metrics via Prometheus

## Build
```shell
$ mkdir bin
$ go build -C src/ -o ../bin/. -v
```

## Usage
### Receiver
Create config file:
```shell
$ tee config.yaml << __EOF__
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
    group_address: "239.0.0.2"
    port: 5000
  - name: "Channel 2 - A feed"
    interface: "wlp0s20f3"
    group_address: "239.0.1.1"
    port: 5001
  - name: "Channel 2 - B feed"
    interface: "wlp0s20f3"
    group_address: "239.0.1.2"
    port: 5002
__EOF__
```
Specify the interface, where you want to receive multicast packets, and the multicast groups you want to listen to.

Start the receiver:
```shell
$ ./bin/multicast-tester receiver ./receiver/config.yaml
```

### Sender
Pass the same config file to the sender:
```shell
$ ./bin/multicast-tester sender \
  [ff03::1]:5000@wlp0s203 \
  239.0.0.1:5000@wlp0s20f3 \
  [ff03::2]:5000@wlp0s20f3 \
  239.0.1.1:5001@wlp0s20f3
```
