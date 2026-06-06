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
- TLS connection validation and certificate inspection (STARTTLS, custom trust stores, weak algorithm detection)
- Traceroute based on a system installed mtr (not available on Windows)
- Query basic IP information from [https://ifconfig.is](https://ifconfig.is).
- Echo Server and Client
- also available as docker container

## Contents

- [Installation](#installation)
- [Global flags](#global-flags)
- [icmp — Ping using ICMP protocol](#icmp--ping-using-icmp-protocol)
- [tcp — Ping using TCP protocol](#tcp--ping-using-tcp-protocol)
- [http — HTTP trace](#http--http-trace)
- [tls — Validate TLS connection or certificate](#tls--validate-tls-connection-or-certificate)
- [mtr — Traceroute using MTR](#mtr--traceroute-using-mtr)
- [query — Query host IP information](#query--query-host-ip-information)
- [echo — Echo server and client](#echo--echo-server-and-client)
- [version — Print version information](#version--print-version-information)
- [Credits](#credits)

---

## Installation

Download latest release binaries from [Github Releases](https://github.com/Tommi2Day/tcping2/releases)
or use released Docker Container on [Dockerhub](https://hub.docker.com/r/tommi2day/tcping2)
```
docker pull tommi2day/tcping2
```
## build Docker Container
```
docker build -t tcping2 -f Dockerfile .
```
container exposes port 8080 for echo server

---

## Global flags

These flags apply to every command:

| Flag | Description |
|------|-------------|
| `--debug` | Verbose debug output |
| `--info` | Reduced info output |
| `--no-color` | Disable colored log output |
| `--dnsIPv4` | Return only IPv4 addresses from DNS |
| `--dnsServer string` | DNS server IP address to query |
| `--dnsPort int` | DNS server port |
| `--dnsTCP` | Query DNS with TCP instead of UDP |
| `--dnsTimeout int` | DNS timeout in seconds |

---

## icmp — Ping using ICMP protocol

```sh
tcping2 icmp [--address <host>] [global flags]
```

> **Note:** Root permission is required (raw socket). Use `sudo` or set the setuid bit on the binary.

| Flag | Description |
|------|-------------|
| `-a, --address string` | IP/host to ping |

**Examples:**

```sh
# Ping google.com (IPv4 only)
sudo tcping2 icmp -a google.com --dnsIPv4
ICMP   OPEN      74.125.133.138    16.9 ms
ICMP   OPEN      74.125.133.101    16.9 ms
ICMP   OPEN      74.125.133.100    17.1 ms

# Ping google.com (IPv4 + IPv6)
sudo tcping2 icmp -a google.com
ICMP   OPEN      142.250.185.238    10.3 ms
ICMP   ERROR     2a00:1450:4001:82f::200e
```

---

## tcp — Ping using TCP protocol

```sh
tcping2 tcp [--address <host>] [--port <port>] [global flags]
```

The address and port can be given as positional arguments, as `host:port` in `--address`, or as separate flags.

| Flag | Description |
|------|-------------|
| `-a, --address string` | IP/host to ping (also accepts `host:port`) |
| `-p, --port string` | TCP port to ping |
| `-t, --timeout int` | Ping timeout in seconds (default 3) |

**Examples:**

```sh
# Equivalent forms
tcping2 tcp google.com 80 --dnsIPv4
tcping2 tcp -a google.com -p 80 --dnsIPv4
tcping2 tcp -a google.com:80 --dnsIPv4
TCP    OPEN      173.194.76.102:80
TCP    OPEN      173.194.76.138:80

# With IPv6 result
tcping2 tcp -a google.com -p 443
TCP    OPEN      142.250.185.238:443
TCP    ERROR: dial tcp [2a00:1450:4001:82f::200e]:443: connect: network is unreachable
```

---

## http — HTTP trace

```sh
tcping2 http --address <url> [global flags]
```

Runs an HTTP trace showing DNS lookup, TCP, TLS, processing, and transfer times.

| Flag | Description |
|------|-------------|
| `-a, --address string` | URL to trace |

**Examples:**

```sh
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
```

---

## tls — Validate TLS connection or certificate

### Validate a connection

```sh
tcping2 tls [--address <host>] [--port <port>] [--rootca <path>] [--starttls <proto>] [global flags]
```

Connects to the server and validates the TLS certificate chain. Uses the system trust store by default. On success it shows the expiry date; on failure it prints the exact reason.

| Flag | Description |
|------|-------------|
| `-a, --address string` | Host (or `host:port`) to connect to |
| `-p, --port int` | TCP port (default `443`) |
| `-r, --rootca string` | Additional trust source: PEM file, directory of PEM/CRT files, Java JKS trust store (`.jks`), or PKCS12 bundle (`.p12`/`.pfx`) |
| `--starttls string` | Upgrade via STARTTLS before TLS handshake: `smtp`, `imap`, `pop3`, `ftp` |
| `-f, --certfile string` | Validate a local certificate file (PEM or DER) instead of connecting |
| `-t, --timeout int` | Connection timeout in seconds (default `5`) |

**Examples:**

```sh
# Validate a public HTTPS server
tcping2 tls -a example.com
TLS    VALID     example.com:443  (expires 2026-10-01, 117 days)

# Invalid / expired certificate
tcping2 tls -a expired.badssl.com
TLS    INVALID   expired.badssl.com:443
      REASON    x509: certificate has expired or is not yet valid: ...

# SMTP with STARTTLS
tcping2 tls -a smtp.gmail.com -p 587 --starttls smtp
TLS    VALID     smtp.gmail.com:587  (expires 2026-08-10, 65 days)

# IMAP with STARTTLS
tcping2 tls -a imap.gmail.com -p 143 --starttls imap
TLS    VALID     imap.gmail.com:143  (expires 2026-08-10, 65 days)

# Custom CA (PEM file)
tcping2 tls -a internal.host -r /etc/ssl/company-ca.pem

# Custom CA (directory of PEM/CRT files)
tcping2 tls -a internal.host -r /etc/ssl/company-certs/

# Java JKS trust store
tcping2 tls -a internal.host -r /opt/jdk/lib/security/cacerts.jks

# PKCS12 trust store
tcping2 tls -a internal.host -r bundle.p12

# Local certificate file — checks validity and expiry
tcping2 tls -f /path/to/server.pem
TLS    VALID     /path/to/server.pem  (expires 2026-10-01, 117 days)

# Weak signature algorithm warning (SHA-1)
tcping2 tls -f /path/to/old-cert.pem
TLS    VALID     /path/to/old-cert.pem  (expires 2026-10-01, 117 days)
       WARN      /path/to/old-cert.pem uses a weak signature algorithm (SHA1-RSA)
```

### Show certificate details

```sh
tcping2 tls show [--address <host>] [--port <port>] [--chain] [--starttls <proto>] [global flags]
```

Connects and prints the leaf certificate's subject, issuer, signature algorithm, SANs, and validity window. Use `--chain` to display every certificate in the peer chain.

| Flag | Description |
|------|-------------|
| `-a, --address string` | Host (or `host:port`) to connect to |
| `-p, --port int` | TCP port (default `443`) |
| `-r, --rootca string` | Additional trust source (same as `tls`) |
| `--starttls string` | STARTTLS protocol (same as `tls`) |
| `--chain` | Show the full certificate chain |
| `-t, --timeout int` | Connection timeout in seconds (default `5`) |

**Examples:**

```sh
tcping2 tls show -a www.google.com
TLS    CERT      www.google.com:443
  Subject:       CN=www.google.com
  Issuer:        CN=WE2,O=Google Trust Services,C=US
  Signature:     ECDSA-SHA256
  Not Before:    2026-05-18 18:38:16 UTC
  Not After:     2026-08-10 18:38:15 UTC  (65 days)
  SANs:          www.google.com
  Serial:        f666bd12cd02a3cb / SN 327523536107629...

tcping2 tls show -a www.google.com --chain
TLS    CERT      www.google.com:443
  Subject:       CN=www.google.com
  ...
  Chain[1]:
    Subject:       CN=WE2,O=Google Trust Services,C=US
    Signature:     ECDSA-SHA384
    Not After:     2029-02-20 14:00:00 UTC  (990 days)
    ...
  Chain[2]:
    Subject:       CN=GTS Root R4,O=Google Trust Services LLC,C=US
    Signature:     SHA256-RSA
    ...

# SHA-1 certificate flagged in show output
tcping2 tls show -a legacy.example.com
TLS    CERT      legacy.example.com:443
  Subject:       CN=legacy.example.com
  Signature:     SHA1-RSA  [WEAK]
  ...
```

---

## mtr — Traceroute using MTR

```sh
tcping2 mtr --address <host> [--tcp] [--port <port>] [global flags]
```

Runs a traceroute via the system-installed `mtr` binary. Not available on Windows.

> **Note:** Root permission is required for ICMP mode. Use `sudo` or set the setuid bit on the binary.

| Flag | Description |
|------|-------------|
| `-a, --address string` | IP/host to trace |
| `-p, --port string` | TCP port (used with `--tcp`) |
| `-t, --tcp` | Use TCP instead of ICMP |
| `-m, --mtr string` | Path to `mtr` binary, or set `MTR_BIN` env var (default `mtr`) |

**Examples:**

```sh
# ICMP trace to google.com
sudo tcping2 mtr -a google.com
Waiting for MTR results to 142.250.184.238 ...
Hop    1 192.168.0.22                   Loss:   0.00% Avg:  0.54
Hop    2 192.168.0.1                    Loss:   0.00% Avg:  1.19
Hop    3 ...                            Loss:   0.00% Avg:  1.66
Hop    9 fra02s19-in-f14.1e100.net      Loss:   0.00% Avg:  9.61

# TCP trace (IPv4 only)
tcping2 mtr -a https://google.com -t --dnsIPv4
Waiting for MTR results to 142.250.185.206:443 ...
Hop    1 192.168.0.22                   Loss:   0.00% Avg:  0.54
Hop   10 fra16s52-in-f14.1e100.net      Loss:   0.00% Avg:  9.83ms
```

---

## query — Query host IP information

```sh
tcping2 query --address <host> [global flags]
```

Queries basic IP geolocation and ASN information from [https://ifconfig.is](https://ifconfig.is).

| Flag | Description |
|------|-------------|
| `-a, --address string` | IP/host to query |

**Examples:**

```sh
tcping2 query -a google.com
IP       :    173.194.76.139
Continent:    North America
Country  :    United States
City     :    Mountain View
Latitude :    37.422000
Longitude:    -122.084000
ASN      :    15169
ORG      :    Google LLC

IP       :    2a00:1450:400c:c06::8b
Continent:    Europe
Country  :    Belgium
City     :    Brussels
ASN      :    15169
ORG      :    Google LLC
```

---

## echo — Echo server and client

```sh
tcping2 echo [--address <host>] [--port <port>] [--server] [global flags]
```

TCP echo server or client. Useful for testing connectivity to ports not yet serving a protocol. The server terminates on a `QUIT\n` message or when `--server-timeout` expires.

| Flag | Description |
|------|-------------|
| `-a, --address string` | IP/host to contact (client mode) |
| `-p, --port string` | TCP port to contact/serve |
| `-s, --server` | Run as echo server |
| `-T, --server-timeout int` | Server timeout in seconds (default 60) |
| `-t, --timeout int` | Client timeout in seconds (default 3) |

**Examples:**

```sh
# Start echo server
tcping2 echo --server -p 8080

# Connect with echo client
tcping2 echo localhost 8080
connection to 127.0.0.1:8080 successful tested

# Test with nc
echo -e "Hello\nQUIT\n" | nc localhost 8080
Hello

# Docker: start echo server in background, then connect
docker run -d --rm -p 8080:8080 docker-prod.hv.devk.de/dba-cloud/goproj/tcping2:1.1.2
tcping2 echo localhost 8080

# Docker: run a one-off TCP ping without local installation
docker run -it --rm docker-prod.hv.devk.de/dba-cloud/goproj/tcping2:1.1.2 tcp google.com 80 --dnsIPv4

# Echo to a standard server (expect timeout)
tcping2 echo www.google.com:80 --timeout 3
Error: failed to read data, err: read tcp 127.0.0.1:65324->172.217.23.100:80: i/o timeout
```

---

## version — Print version information

```sh
tcping2 version
```

Prints the build version, commit hash, and build date.

---

## Credits

- [lmas/icmp_ping.go](https://gist.github.com/lmas/c13d1c9de3b2224f9c26435eb56e6ef3)
- [sparrc/go-ping](https://github.com/sparrc/go-ping)
- [davecheney/httpstat](https://github.com/davecheney/httpstat)
- [i3h/tcping](https://github.com/i3h/tcping)
