# forward
Reads Stdin and forwards data to a remote destination over UDP/TCP/TCP+TLS.  
Defaults to connecting via TCP+TLS and teeing input to stdout

Data is buffered until reaching a newline char before being sent over the network.  Should the connection fail to connect or the connection is lost, `forward` will not panic so the superceding process can continue uninterrupted.

Future work:  
Possibly reconnect TCP connection if lost.

### Usage
```shell
forward [OPTIONS] host:port
Options:
  -t, --tee=true: Tee stdin to stdout
  -s, --tls=true: Connect with TLS
  -u, --udp=false: Send via UDP (will ignore TLS)
```

### Example
```shell
❯❯❯ echo "Test Log Message" | forward logs3.papertrailapp.com:XXXXX

# To capture stderr as well
# Note: Stderr captured through `2>&1|` will be printed to stdout when using the tee option.
❯❯❯ ./std_generator 2>&1| forward logs3.papertrailapp.com:XXXXX
```

### Recommended calls
```shell
❯❯❯ set -o pipefail  # return code will any non-zero returned from any commands in the pipe
```

### Build
```shell
❯❯❯ go get github.com/NickSardo/forward
```
