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
const serverTimeout = 60

type echoResult struct {
	err    error
	finish bool
}

var echoTimeout = 60

func init() {
	echoCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "ip/host to contact")
	echoCmd.Flags().StringVarP(&queryPort, "port", "p", "", "tcp port to contact/serve")
	echoCmd.Flags().IntVarP(&pingTimeout, "timeout", "t", pingTimeout, "Echo Timeout in sec")
	echoCmd.Flags().IntVarP(&echoTimeout, "server-timeout", "T", echoTimeout, "Echo Server Timeout in sec")
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
		if echoTimeout < serverTimeout {
			echoTimeout = serverTimeout
		}
		log.Debugf("set server timeout to %d seconds", echoTimeout)
		ip := common.GetEnv("ECHO_SERVER", "")
		resCh := make(chan echoResult)
		defer close(resCh)
		go runEchoServer(ip, queryPort, resCh)
		select {
		case r := <-resCh:
			return r.err
		case <-time.After(time.Duration(echoTimeout) * time.Second):
			err = fmt.Errorf("timeout")
			return err
		}
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
func runEchoServer(host, port string, resCh chan echoResult) {
	// create a tcp listener on the given port
	log.Debugf("try to listen on %s:%s", host, port)

	addr := net.JoinHostPort(host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Errorf("failed to create listener:%s", err)
		resCh <- echoResult{err: err, finish: true}
		return
	}
	fmt.Printf("listening on %s, terminate with CTRL-C\n", listener.Addr())
	ch := make(chan echoResult)
	defer close(ch)
	// listen for new connections
	for {
		conn, e := listener.Accept()
		if e != nil {
			err = fmt.Errorf("failed to accept connection:%s", e)
			resCh <- echoResult{err: err, finish: true}
			return
		}

		// pass an accepted connection to a handler goroutine
		go handleServerConnection(conn, ch)
		r := <-ch
		if r.finish {
			resCh <- r
			break
		}
	}
}

// handleConnection handles the lifetime of a connection
func handleServerConnection(conn net.Conn, ch chan echoResult) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	reader := bufio.NewReader(conn)
	version := GetVersion(false)
	servername := common.GetHostname()
	remote := conn.RemoteAddr().String()
	msg := fmt.Sprintf("got connection from %s", remote)
	log.Info(msg)
	fmt.Println(msg)
	log.Debugf("set server connection deadline to %d seconds", pingTimeout)
	for {
		// read client request data
		_ = conn.SetDeadline(time.Now().Add(time.Duration(pingTimeout) * time.Second))
		time.Sleep(200 * time.Millisecond)
		bytes, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			switch {
			case err == io.EOF:
				log.Debug("EOF")
				fmt.Println("No Data from client")
				ch <- echoResult{err: err, finish: false}
				return
			case strings.Contains(err.Error(), "i/o timeout"):
				err = fmt.Errorf("IO Timeout on server")
				log.Debug(err.Error())
				fmt.Println(err.Error())
				ch <- echoResult{err: err, finish: false}
				return
			case strings.Contains(err.Error(), "wsarecv"):
				err = fmt.Errorf("connection closed by client")
				log.Debug(err.Error())
				fmt.Println(err.Error())
				ch <- echoResult{err: err, finish: false}
				return
			default:
				err = fmt.Errorf("failed to read data from client, err:%s", err)
				log.Debugf("%s, terminate", err)
				ch <- echoResult{err: err, finish: true}
				return
			}
		}
		msg = strings.TrimSuffix(string(bytes), "\n")
		log.Infof("got %s from client", msg)
		if strings.HasPrefix(msg, echoPrefix) {
			log.Debugf("prefix is %s, send version to client", echoPrefix)
			amsg := fmt.Sprintf("%s Server %s %s\n", echoPrefix, servername, version)
			_, err = conn.Write([]byte(amsg))
			if err != nil {
				log.Debugf("failed to write data to client, err:%s", err)
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
		fmt.Println("got ", msg)
		// echo the message back to the client
		log.Debugf("echo '%s' back to client", msg)
		_, err = conn.Write(bytes)
		if err != nil {
			log.Debugf("failed to return data to client, err:%s", err)
			ch <- echoResult{err: err, finish: true}
			return
		}
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
	if len(ips) > 1 {
		log.Warnf("more than one IP address found, using the first one")
	}
	// get the first IP address and create an address string
	ip := ips[0].String()
	version := GetVersion(false)
	servername := common.GetHostname()
	addr := net.JoinHostPort(ip, queryPort)

	// create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(echoTimeout)*time.Second)
	defer cancel()
	log.Debugf("set client timeout to %d seconds", echoTimeout)

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

	// set a deadline for the connection read/write operations double
	dl := pingTimeout * 2
	log.Debugf("double connection timeout to %d seconds", dl)
	_ = conn.SetDeadline(time.Now().Add(time.Duration(dl) * time.Second))

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
		err = fmt.Errorf("failed to read data from server, err:%s", err)
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
