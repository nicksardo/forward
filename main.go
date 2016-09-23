package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
)

var (
	useUDP bool
	useTLS bool
	doTee  bool

	syslog             bool
	syslogHostname     string
	syslogApp          string
	syslogPriority     int
	syslogAttachHeader bool
)

func main() {
	app := cli.NewApp()
	app.Name = "forward"
	app.Usage = "Transport StdIn lines to a remote destination over UDP, TCP, or TCP+TLS"
	app.UsageText = "forward [global options] [syslog [syslog options]] address:port"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "udp, u",
			Usage:       "Send via UDP (will ignore TLS)",
			Destination: &useUDP,
		},
		cli.BoolTFlag{
			Name:        "tls, s",
			Usage:       "TLS-secured TCP connection",
			Destination: &useTLS,
		},
		cli.BoolTFlag{
			Name:        "tee, t",
			Usage:       "Tee stdin to stdout",
			Destination: &doTee,
		},
	}
	app.Action = func(c *cli.Context) error {
		forward(c.Args().First())
		return nil
	}

	h, _ := os.Hostname()
	app.Commands = []cli.Command{
		{
			Name:      "syslog",
			Aliases:   []string{"log"},
			Usage:     "Wrap lines in RFC-5424 Syslog format",
			ArgsUsage: "address:port",
			Action: func(c *cli.Context) error {
				syslog = true
				forward(c.Args().First())
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "hostname, n",
					Value:       h,
					Destination: &syslogHostname,
				},
				cli.StringFlag{
					Name:        "app, a",
					Value:       "logger",
					Destination: &syslogApp,
				},
				cli.IntFlag{
					Name:        "priority, p",
					Value:       22,
					Destination: &syslogPriority,
				},
				cli.BoolFlag{
					Name:        "att, x",
					Usage:       "Attach header to message",
					Destination: &syslogAttachHeader,
				},
			},
		},
	}

	app.Run(os.Args)
}

func forward(destination string) {
	if !validDestination(destination) {
		return
	}
	var conn net.Conn
	var err error
	switch {
	case useUDP:
		conn, err = net.Dial("udp", destination)
	case !useTLS:
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
	if doTee {
		stdOutReader, stdOutWriter := io.Pipe()
		writers = append(writers, stdOutWriter)
		go func() {
			io.Copy(os.Stdout, stdOutReader)
		}()
	}
	if connected {
		netReader, netWriter := io.Pipe()
		writers = append(writers, netWriter)
		go func() {
			wg.Add(1)
			reader := bufio.NewReader(netReader)
			byteBuffer := bytes.NewBuffer([]byte{})
			for {
				data, err := reader.ReadBytes('\n')
				if err != nil {
					break
				}

				if syslog {
					if syslogAttachHeader {
						data = toSyslog(byteBuffer, data)
					} else {
						byteBuffer.Write(data)
						data = byteBuffer.Bytes()
					}
				}

				if _, err = conn.Write(data); err != nil {
					break
				}

				if syslog {
					byteBuffer.Reset()
				}
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

func toSyslog(b *bytes.Buffer, line []byte) []byte {
	//<22>1 2016-06-18T09:56:21Z sendername programname - - - the log message
	b.WriteString("<")
	b.WriteString(strconv.Itoa(syslogPriority))
	b.WriteString(">1 ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339) + " ")
	b.WriteString(syslogHostname + " ")
	b.WriteString(syslogApp + " ")
	b.WriteString("- - - ")
	b.Write(line)

	return b.Bytes()
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
