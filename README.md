# tcping2

[![Go Report Card](https://goreportcard.com/badge/github.com/tommi2day/tcping2)](https://goreportcard.com/report/github.com/tommi2day/tcping2)
![CI](https://github.com/tommi2day/tcping2/actions/workflows/main.yml/badge.svg)
[![codecov](https://codecov.io/gh/Tommi2Day/tcping2/branch/main/graph/badge.svg?token=C1IP9AMBUM)](https://codecov.io/gh/Tommi2Day/tcping2)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/tcping2)
[![Docker Pulls](https://img.shields.io/docker/pulls/tommi2day/tcping2.svg)](https://hub.docker.com/r/tommi2day/tcping2/)


Tcping2 is an ip probe command line tool, supporting ICMP, TCP and HTTP protocols and echo server and client

## Features

- Support ICMP/TCP protocols
- Support resolving hostnames to IPv4/IPv6 addresses or IPv4 Only
- HTTPTrace
- Traceroute based on a system installed mtr (not available on Windows)
- Query basic IP information from [https://ifconfig.is](https://ifconfig.is).
- Echo Server and Client
- also available as docker container

## Installation

Download latest release binaries from [Github Releases](https://github.com/Tommi2Day/tcping2/releases)
or use released Docker Container on [Dockerhub](https://hub.docker.com/r/tommi2day/tcping2)
```
docker pull tommi2day/tcping2
```
or build docker container for yourself
```
docker build -t tcping2 -f Dockerfile .
```
container exposes port 8080 for echo server

or Build on your own
```
git clone https://github.com/tommi2day/tcping2.git
go build
``` 
## Usage

```
tcping2 --help
  Usage:
  tcping2 [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  echo        try echo using TCP protocol
  help        Help about any command
  http        Ping using HTTP protocol
  icmp        Ping using ICMP protocol
  mtr         Traceroute using MTR
  query       Query host ip information
  tcp         Ping using TCP protocol
  version     version print version string

Flags:
      --debug              verbose debug output
      --dnsIPv4            return only IPv4 Addresses from DNS Server
      --dnsPort int        DNS Server Port Address
      --dnsServer string   DNS Server IP Address to query
      --dnsTCP             Query DNS with TCP instead of UDP
      --dnsTimeout int     DNS Timeout in sec
  -h, --help               help for tcping2
      --info               reduced info output
      --no-color           disable colored log output
      --unit-test          redirect output for unit tests

Use "tcping2 [command] --help" for more information about a command.
#-------------------------------------------------------------
tcping2 echo --help
Usage:
  tcping2 echo [flags]

Flags:
  -a, --address string       ip/host to contact
  -h, --help                 help for echo
  -p, --port string          tcp port to contact/serve
  -s, --server               Run as echo server
  -T, --server-timeout int   Echo Server Timeout in sec (default 60)
  -t, --timeout int          Echo Timeout in sec (default 3)
#-------------------------------------------------------------
tcping2 http --help
Run httptrace to the target

Usage:
  tcping2 http [flags]

Flags:
  -a, --address string   URL to query
  -h, --help             help for http
#-------------------------------------------------------------
tcping2 icmp --help
Ping using ICMP protocol

Usage:
  tcping2 icmp [flags]

Flags:
  -a, --address string   ip/host to query
  -h, --help             help for icmp
#-------------------------------------------------------------
tcping2 tcp --help
Ping using TCP protocol

Usage:
  tcping2 tcp [flags]

Flags:
  -a, --address string   ip/host to ping
  -h, --help             help for tcp
  -p, --port string      tcp port to ping
  -t, --timeout int      Ping Timeout in sec (default 3)
#-------------------------------------------------------------
tcping2 mtr --help
Traceroute using MTR

Usage:
  tcping2 mtr [flags]

Flags:
  -a, --address string   ip/host to ping
  -h, --help             help for mtr
  -m, --mtr string       mtr binary path or use MTR_BIN env var (default "mtr")
  -p, --port string      tcp port to ping
  -t, --tcp              use TCP instead of ICMP


#-------------------------------------------------------------
tcping2 query --help
Query host ip information

Usage:
  tcping2 query [flags]

Flags:
  -a, --address string   ip/host to query
  -h, --help             help for query
#-------------------------------------------------------------
tcping2 version
```
### use docker container
docker container can be used to run tcping2 without installation. Per default it starts the echo server. 
```bash
docker run -d --rm -p 8080:8080 tommi2day/tcping2
listening on [::]:8080, terminate with CTRL-C
...
echo -e "Hello\nQUIT\n"|nc localhost 8080
Hello
#-------------------------------------------------------------
docker run -it --rm tommi2day/tcping2 tcp google.com 80 --dnsIPv4
```

### Note

Root permission is required when running ICMP ping or MTR, since it needs to open raw socket.
You can either use sudo command, or set setuid bit for tcping2.

## Examples

```bash
# ping google.com (ipv4 only)
sudo tcping2 icmp google.com --dnsIPv4
ICMP   OPEN      74.125.133.138    16.9 ms
ICMP   OPEN      74.125.133.101    16.9 ms
ICMP   OPEN      74.125.133.100    17.1 ms
ICMP   OPEN      74.125.133.102    16.7 ms
ICMP   OPEN      74.125.133.113    16.7 ms
ICMP   OPEN      74.125.133.139    16.8 ms
#-------------------------------------------------------------
tcping2 icmp -a google.com
ICMP   OPEN      142.250.185.238    10.3 ms
ICMP   ERROR     2a00:1450:4001:82f::200e
#-------------------------------------------------------------
# ping google.com  with port given (mandantory)
tcping2 tcp google.com 80 --dnsIPv4 # or
tcping2 tcp -a google.com -p 80 --dnsIPv4 # or
tcping2 tcp -a google.com:80 --dnsIPv4
TCP    OPEN      173.194.76.102:80
TCP    OPEN      173.194.76.138:80
TCP    OPEN      173.194.76.101:80
TCP    OPEN      173.194.76.139:80
TCP    OPEN      173.194.76.100:80
TCP    OPEN      173.194.76.113:80
#-------------------------------------------------------------
tcping2 tcp -a google.com -p 443
TCP    OPEN      142.250.185.238:443
TCP    ERROR: dial tcp [2a00:1450:4001:82f::200e]:443: connect: network is unreachable
#-------------------------------------------------------------
# http trace google.com
tcping2 http -a google.com
URL       :    https://google.com
Proxy     :    false
Scheme    :    https
Host      :    google.com
Port      :    443
DNS Lookup:    26.54 ms
TCP       :    17.29 ms
TLS       :    21.84 ms
Process   :    50.68 ms
Transfer  :    0.11 ms
Total     :    116.56 ms
#-------------------------------------------------------------
# query ip informations for google.com
tcping2 query -a google.com
IP       :    173.194.76.139
Continent:    North America
Country  :    United States
City     :    Mountain View
Latitude :    37.422000
Longitude:    -122.084000
ASN      :    15169
ORG      :    Google LLC

IP       :    173.194.76.138
Continent:    North America
Country  :    United States
City     :    Mountain View
Latitude :    37.422000
Longitude:    -122.084000
ASN      :    15169
ORG      :    Google LLC
....
IP       :    2a00:1450:400c:c06::8b
Continent:    Europe
Country  :    Belgium
City     :    Brussels
Latitude :    50.850300
Longitude:    4.351710
ASN      :    15169
ORG      :    Google LLC
#-------------------------------------------------------------

# run ip icmp trace to google.com
sudo tcping2 mtr -a google.com
Waiting for MTR results to 142.250.184.238 ...
Hop    1 192.168.0.22                                                 Loss:   0.00% Avg:  0.54
Hop    2 192.168.0.1                                                  Loss:   0.00% Avg:  1.19
Hop    3 ...                                                          Loss:   0.00% Avg:  1.66
Hop    4 ...                                                          Loss:   0.00% Avg:  5.61
Hop    5 ...                                                          Loss:   0.00% Avg:  9.66
Hop    6 ...                                                          Loss:   0.00% Avg:  9.53
Hop    7 209.85.142.109                                               Loss:   0.00% Avg: 12.30
Hop    8 172.253.66.139                                               Loss:   0.00% Avg:  9.81
Hop    9 fra02s19-in-f14.1e100.net                                    Loss:   0.00% Avg:  9.61
Waiting for MTR results to 2a00:1450:4001:82b::200e ...
MTR    exit status 1:mtr: udp socket connect failed: Network is unreachable
#-------------------------------------------------------------

# run tcp trace to google.com (IPv4 only)
tcping2 mtr -a https://google.com -t --dnsIPv4
Waiting for MTR results to 142.250.185.206:443 ...
Hop    1 192.168.0.22                                                 Loss:   0.00% Avg:  0.54
Hop    2 192.168.0.1                                                  Loss:   0.00% Avg:  1.19
Hop    3 ...                                                          Loss:   0.00% Avg:  1.66
Hop    4 ...                                                          Loss:   0.00% Avg:  5.61
Hop    5 ...                                                          Loss:   0.00% Avg:  9.66
Hop    6 ???                                                          Loss: 100.00% Avg:  0.00ms
Hop    7 192.178.71.154                                               Loss:   0.00% Avg:  9.85ms
Hop    8 209.85.244.249                                               Loss:   0.00% Avg: 10.54ms
Hop    9 142.250.225.77                                               Loss:   0.00% Avg:  9.75ms
Hop   10 fra16s52-in-f14.1e100.net                                    Loss:   0.00% Avg:  9.83ms

#-------------------------------------------------------------

# start echo server
# `QUIT\n` terminates the server
tcping2 echo --server -p 8080 --timeout 10
# or use docker container, which starts without parameter the echo server (or give the full command parameters)
docker run -it --rm -p 8080:8080 tommi2day/tcping2
#connect with  tcping as echo client
tcping2 echo localhost 8080
# server output
listening on [::]:8080, terminate with CTRL-C
got connection from 172.17.0.1:37806
got  TCPING2 , client localhost tcping2 version 0.0.1-beta (d8f3af8 - 2024-04-28)
got connection from 172.17.0.1:37806
got quit, terminate server
# client output
connection to 127.0.0.1:8080 successful tested
#-------------------------------------------------------------
#Test with echo server and nc as client
echo -e "Hello\nQUIT\n"|nc localhost 8080
Hello
# server
docker run -it --rm -p 8080:8080 tommi2day/tcping2
listening on [::]:8080, terminate with CTRL-C
got connection from 172.17.0.1:41918
got  Hello
got connection from 172.17.0.1:41918
IO Timeout
#-------------------------------------------------------------
# echo to standard server with timeout
tcping2 echo www.google.com:80 --timeout 3
Error: failed to read data, err:read tcp 127.0.0.1:65324->172.217.23.100:80: i/o timeout
```

## Credits

- [lmas/icmp_ping.go](https://gist.github.com/lmas/c13d1c9de3b2224f9c26435eb56e6ef3)
- [sparrc/go-ping](https://github.com/sparrc/go-ping)
- [davecheney/httpstat](https://github.com/davecheney/httpstat)
- [i3h/tcping](https://github.com/i3h/tcping)
