# forward
Forwards StdIn data to a remote destination over UDP/TCP/TCP+TLS.  
Defaults to connecting via TCP+TLS and teeing input to stdout

Data is buffered until reaching a newline char before being sent over the network.  Should the connection fail to connect or the connection is lost, `forward` will not panic so the upstream process can continue uninterrupted.

Future work:  
Possibly reconnect TCP connection if lost.

### Usage
```
NAME:
   forward - Transport StdIn lines to a remote destination over UDP, TCP, or TCP+TLS

USAGE:
   forward [global options] [syslog [syslog options]] address:port

VERSION:
   0.1

COMMANDS:
   syslog, log	Wrap lines in RFC-5424 Syslog format
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --udp, -u		Send via UDP (will ignore TLS)
   --tls, -s		TLS-secured TCP connection
   --tee, -t		Tee stdin to stdout
   --help, -h		show help
   --version, -v	print the version
```

###### Syslog Usage
```
NAME:
   forward syslog - Wrap lines in RFC-5424 Syslog format

USAGE:
   forward syslog [command options] address:port

OPTIONS:
   --hostname, -n "MBP.local"    # Uses local hostname by default
   --app, -a "logger"           
   --priority, -p "22"
```

### Example
```shell
❯❯❯ echo "Test Log Message" | forward logs3.papertrailapp.com:XXXXX

# To capture stderr as well
# Note: Stderr captured through `2>&1|` will be printed to stdout when using the tee option.
❯❯❯ ./std_generator 2>&1| forward logs3.papertrailapp.com:XXXXX
```

##### Wrapping lines with syslog format
```shell
❯❯❯ echo "Test Log Message" | forward log -n some.host.name -a worker -p 15 logs3.papertrailapp.com:XXXXX
```
yields
```syslog
<15>1 2016-02-29T09:22:48Z some.host.name worker - - - Test Log Message
```


### Recommended Settings
```shell
❯❯❯ set -o pipefail  # return code will any non-zero returned from any commands in the pipe
```

### Build
```shell
❯❯❯ go get github.com/NickSardo/forward
```

### Why?
Papertrail's [recommended approach](https://github.com/papertrail/remote_syslog2/issues/49) is to use `logger` with `rsyslogd`.  For the following reasons, it may be beneficial to use this instead.

1.  You're using a small image of Alpine linux which doesn't have rsyslogd pre-installed
1.  You need security but your version of `netcat` doesn't support TLS
1.  You just want to log something quickly without syslog config
