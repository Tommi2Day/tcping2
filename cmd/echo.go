package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	echoCmd = &cobra.Command{
		Use:   "echo",
		Short: "try echo using TCP protocol",
		Long: `this command will try to get an echo on a message to a server using TCP protocol
and can act as Server or client`,
		RunE:         runEcho,
		SilenceUsage: true,
	}
	echoServer = false
)

const echoPrefix = "TCPING2"
const echoQuit = "QUIT"

type echoResult struct {
	err    error
	finish bool
}

func init() {
	echoCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "ip/host to contact")
	echoCmd.Flags().StringVarP(&queryPort, "port", "p", "", "tcp port to contact/serve")
	echoCmd.Flags().IntVarP(&pingTimeout, "timeout", "t", pingTimeout, "Echo Timeout in sec")
	echoCmd.Flags().BoolVarP(&echoServer, "server", "s", false, "Run as echo server")
	RootCmd.AddCommand(echoCmd)
}

func runEcho(_ *cobra.Command, args []string) error {
	var err error
	log.Debug("Echo called")
	// get arguments
	if echoServer {
		if len(args) > 0 && queryPort == "" {
			queryPort = args[0]
		}
		if queryPort == "" {
			return fmt.Errorf("please specify a port to serve on")
		}
		ip := common.GetEnv("ECHO_SERVER", "")
		err = runEchoServer(ip, queryPort)
		return err
	}
	if len(args) > 0 && queryAddress == "" {
		queryAddress = args[0]
	}
	if len(args) > 1 && queryPort == "" {
		queryPort = args[1]
	}
	if queryAddress == "" {
		return fmt.Errorf("please specify an address to query")
	}
	err = runClient()
	log.Info("Echo done")
	return err
}
func runEchoServer(host, port string) (err error) {
	// create a tcp listener on the given port
	log.Debugf("try to listen on %s:%s", host, port)
	ch := make(chan echoResult)
	addr := net.JoinHostPort(host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Errorf("failed to create listener:%s", err)
		return
	}
	fmt.Printf("listening on %s, terminate with CTRL-C\n", listener.Addr())

	// listen for new connections
	for {
		conn, e := listener.Accept()
		if e != nil {
			err = fmt.Errorf("failed to accept connection:%s", e)
			return
		}

		// pass an accepted connection to a handler goroutine
		go handleServerConnection(conn, ch)
		r := <-ch
		err = r.err
		if r.finish {
			break
		}
	}
	return
}

// handleConnection handles the lifetime of a connection
func handleServerConnection(conn net.Conn, ch chan echoResult) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	defer close(ch)

	log.Debugf("set connection deadline to %d seconds", pingTimeout)
	reader := bufio.NewReader(conn)
	version := GetVersion(false)
	servername := common.GetHostname()

	for {
		// read client request data
		_ = conn.SetDeadline(time.Now().Add(time.Duration(pingTimeout) * time.Second))
		remote := conn.RemoteAddr().String()
		msg := fmt.Sprintf("got connection from %s", remote)
		log.Infof(msg)
		fmt.Println(msg)
		bytes, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			if err != io.EOF {
				err = fmt.Errorf("failed to read data, err:%s", err)
			}
			log.Error(err)
			ch <- echoResult{err: err, finish: true}
			return
		}
		msg = strings.TrimSuffix(string(bytes), "\n")
		log.Infof("got %s from client", msg)
		if strings.HasPrefix(msg, echoPrefix) {
			log.Debugf("prefix is %s, send version to client", echoPrefix)
			amsg := fmt.Sprintf("%s Server %s %s\n", echoPrefix, servername, version)
			_, err = conn.Write([]byte(amsg))
			if err != nil {
				log.Errorf("failed to write data to client, err:%s", err)
				ch <- echoResult{err: err, finish: true}
				return
			}
		}
		if strings.HasPrefix(msg, echoQuit) {
			log.Debugf("got quit, terminate server")
			fmt.Println("got quit, terminate server")
			ch <- echoResult{err: nil, finish: true}
			return
		}
		fmt.Println(msg)
	}
}

func runClient() (err error) {
	var d net.Dialer
	// obtain the server address and port via program arguments
	log.Debugf("try Echo to %s:%s", queryAddress, queryPort)
	ips, err := normalizeAddress()
	if err != nil {
		err = fmt.Errorf("normalizeAddress failed: %v", err)
		return
	}
	if len(ips) == 0 {
		err = fmt.Errorf("no IP addresses found")
		return
	}
	// get the first IP address and create an address string
	ip := ips[0].String()
	version := GetVersion(false)
	servername := common.GetHostname()
	addr := net.JoinHostPort(ip, queryPort)

	// create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pingTimeout)*time.Second)
	defer cancel()
	log.Debugf("set context timeout to %d seconds", pingTimeout)

	// create a tcp connection to the server
	log.Debugf("connecting to %s", addr)
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		log.Infof("failed to connect :%s", err)
		err = fmt.Errorf("failed to connect to server %s", addr)
		return
	}
	log.Infof("connected to %s", conn.RemoteAddr())
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	// set a deadline for the connection read/write operations
	_ = conn.SetDeadline(time.Now().Add(time.Duration(pingTimeout) * time.Second))

	// send the TCPING message to the server
	log.Infof("send %s version to server", echoPrefix)
	msg := fmt.Sprintf("%s , client %s %s\n", echoPrefix, servername, version)
	log.Debugf("send %s", msg)
	_, err = conn.Write([]byte(msg))
	if err != nil {
		err = fmt.Errorf("failed to write data to server: %s", err)
		log.Info(err)
		return
	}

	// read from connection and send back to server
	time.Sleep(1 * time.Second)
	reader := bufio.NewReader(conn)
	var line []byte
	line, err = reader.ReadBytes(byte('\n'))
	if err != nil {
		if err == io.EOF {
			log.Infof("No Data from server, but connected")
			err = nil
			fmt.Printf("connection to %s successful tested\n", addr)
			return
		}
		err = fmt.Errorf("failed to read data, err:%s", err)
		return
	}
	msg = strings.TrimSuffix(string(line), "\n")
	log.Debugf("received: %s", msg)

	// check if the server response is a TCPING message
	if strings.HasPrefix(msg, echoPrefix) {
		log.Debugf("is %s, send terminate to server", echoPrefix)
		log.Infof("answer: %s", msg)
		_, _ = conn.Write([]byte(echoQuit + "\n"))
	} else {
		log.Infof("not %s, but connected", echoPrefix)
	}
	// print the final response
	fmt.Printf("connection to %s successful tested\n", addr)
	return
}
