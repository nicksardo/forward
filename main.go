package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	flag "github.com/ogier/pflag"
)

func main() {
	// debug := flag.BoolP("debug", "d", false, "Print ")
	useUDP := flag.BoolP("udp", "u", false, "Send via UDP (will ignore TLS)")
	useTLS := flag.BoolP("tls", "s", true, "Connect with TLS")
	doTee := flag.BoolP("tee", "t", true, "Tee stdin to stdout")
	command := "forward [OPTIONS] host:port"
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", command)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) == 0 || !validDestination(flag.Arg(0)) {
		flag.Usage()
		return
	}

	destination := flag.Arg(0)

	var conn net.Conn
	var err error
	switch {
	case *useUDP:
		conn, err = net.Dial("udp", destination)
	case !*useTLS:
		conn, err = net.Dial("tcp", destination)
	default:
		conn, err = tls.Dial("tcp", destination, &tls.Config{})
	}
	connected := false
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to %s: %v\n", destination, err)
	} else {
		connected = true
		defer conn.Close()
	}

	writers := make([]*io.PipeWriter, 0, 2)
	wg := sync.WaitGroup{}
	if *doTee {
		stdOutReader, stdOutWriter := io.Pipe()
		writers = append(writers, stdOutWriter)
		go func() {
			wg.Add(1)
			io.Copy(os.Stdout, stdOutReader)
			wg.Done()
		}()
	}
	if connected {
		netReader, netWriter := io.Pipe()
		writers = append(writers, netWriter)
		go func() {
			wg.Add(1)
			reader := bufio.NewReader(netReader)
			for {
				data, err := reader.ReadBytes('\n')
				if err != nil {
					break
				}
				conn.Write(data)
			}
			io.Copy(ioutil.Discard, netReader)
			wg.Done()
		}()
	}

	if len(writers) == 0 {
		io.Copy(ioutil.Discard, os.Stdin)
		return
	}

	writeOnly := make([]io.Writer, 0, len(writers))
	for _, w := range writers {
		writeOnly = append(writeOnly, w)
	}
	mw := io.MultiWriter(writeOnly...)
	io.Copy(mw, os.Stdin)
	for _, w := range writers {
		w.Close()
	}

	wg.Wait()
}

// Basic validation test
func validDestination(d string) bool {
	if len(d) == 0 {
		return false
	}

	s := strings.Split(d, ":")
	if len(s) != 2 || len(s[0]) == 0 || len(s[1]) == 0 {
		return false
	}

	if _, err := strconv.Atoi(s[1]); err != nil {
		return false
	}

	return true
}
