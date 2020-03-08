# telnet-client
Tiny telnet client for sending commands to remote server.  
Client does passing authorization and parsing of command result. 

## Example of usage
```Go
package main

import (
	"fmt"
	"github.com/tanjmaxalb/telnet-client"
)

func main() {
	tc := telnet.TelnetClient{
		Address:  "127.0.0.1",
		Login:    "user",
		Password: "P@ssw0rd",
	}
	err := tc.Dial()
	if err != nil {
		fmt.Printf("failed open connect with error = %v\n", err)
		return
	}
	defer tc.Close()

	stdout, err := tc.Execute("arp -a")
	if err != nil {
		fmt.Printf("failed execute command with error = %v\n", err)
		return
	}

	fmt.Printf(string(stdout))
}
```

After executing the following content will appear:

```
arp -a 
? (192.168.1.145) at 70:85:c2:6c:e8:a3 [ether]  on br0
? (192.168.1.193) at f8:63:3f:8a:3e:e6 [ether]  on br0
? (192.168.1.216) at 40:e2:30:0e:3e:99 [ether]  on br0
? (192.168.1.162) at b8:27:eb:1c:3c:34 [ether]  on br0
? (192.168.1.144) at 70:85:c2:6c:e8:a3 [ether]  on br0
? (77.106.87.1) at f8:f0:82:75:5a:7d [ether]  on vlan2
? (192.168.1.20) at 94:65:2d:d3:0f:22 [ether]  on br0
? (192.168.1.87) at 70:bb:e9:ee:d4:bb [ether]  on br0
user-77-106-87-53.tomtelnet.ru (77.106.87.53) at f8:f0:82:75:5a:7d [ether]  on vlan2
? (192.168.1.207) at <incomplete>  on br0
```

### Logging

You can set `Verbose` parameter for internal logging and override log stream via `LogWriter` parameter.  
By default, logger write to `os.Stdout`.  

```Go
tc := telnet.TelnetClient{
    Address:  "127.0.0.1",
    Login:    "user",
    Password: "P@ssw0rd",
    Verbose:   true,
	LogWriter: bufio.NewWriter(log.Writer()),
}
```

So you will see following lines:
```
telnet: Trying connect to 192.168.1.1:23
telnet: Waiting for the first banner
telnet: Found login prompt
telnet: Found password prompt
telnet: Send command: arp -a 
telnet: Received data with size = 584
arp -a 
? (192.168.1.145) at 70:85:c2:6c:e8:a3 [ether]  on br0
? (192.168.1.193) at f8:63:3f:8a:3e:e6 [ether]  on br0
...
```
