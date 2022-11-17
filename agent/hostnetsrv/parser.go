package hostnetsrv

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/google/go-cmp/cmp"
)

// /proc/net/tcp (and friends udp, tcp6, udp6) line entries indexes
const (
	lineNum = iota
	localAddress
	remoteAddress
	state
	queue
	transmitTime
	retransmit
	uid
	timeout
	inode
)

const (
	stateListen = "0A"
	serviceType = "HOST"
)

func convertIpFromHex(hex string) string {
	if len(hex) > 8 {
		logger.Warning().Println(pkgName, "IPv6 is not yet supported")
		// fallback to some invalid IPv6 address
		// So it will result to error everywhere, when IPv6 is started testing
		return "ffff::eeee::0001"
	} else {
		var num [4]int64
		var err error
		printOnce := false
		for i := 0; i < 4; i++ {
			num[i], err = strconv.ParseInt(hex[i*2:i*2+2], 16, 8)
			if err != nil && !printOnce {
				// Parse errors here should never happen.
				logger.Error().Println(pkgName, err)
				printOnce = true
			}
		}
		// Hex is in network order. Reverse it.
		return fmt.Sprintf("%d.%d.%d.%d", num[3], num[2], num[1], num[0])
	}
}

// Parse line by line /proc/net/tcp|udp file
// and store only listen state services
func (obj *hostNetServices) parseProcNetFile(name string, services *[]common.ServiceInfoEntry) {
	// true - TCP ports, false = UDP ports
	portTcp := strings.HasPrefix(path.Base(name), "tcp")

	f, err := os.OpenFile(name, os.O_RDONLY, os.ModePerm)
	if err != nil {
		logger.Error().Println(pkgName, name, "open file error: ", err)
		return
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	for {
		entry := common.ServiceInfoEntry{
			IPs: []string{},
			Ports: common.Ports{
				TCP: []uint16{},
				UDP: []uint16{},
			},
		}
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			logger.Error().Println(pkgName, name, "read file error: ", err)
			return
		}
		arr := strings.Fields(line)
		if arr[state] != stateListen {
			continue
		}

		ipPort := strings.Split(arr[localAddress], ":")

		// IP is in hex. Convert to printable string IP format
		entry.IPs = append(entry.IPs, convertIpFromHex(ipPort[0]))

		// Port is in hex. Convert to dec
		port, err := strconv.ParseInt(ipPort[1], 16, 17)
		if err != nil {
			// Parse errors here should never happen.
			logger.Error().Println(pkgName, err)
			port = 0
		}
		if portTcp {
			entry.Ports.TCP = append(entry.Ports.TCP, uint16(port))
		} else {
			entry.Ports.UDP = append(entry.Ports.UDP, uint16(port))
		}

		entry.Name = findNameFromInode(arr[inode])

		*services = append(*services, entry)
	}

}

func findNameFromInode(inode string) string {
	pidsDir, err := filepath.Glob("/proc/[0-9]*/fd/[0-9]*")
	if err != nil {
		logger.Error().Println(pkgName, "parsing /proc dir", err)
		return "Unknown"
	}

	for _, procFd := range pidsDir {
		link, err := os.Readlink(procFd)
		if err != nil {
			continue
		}
		if strings.Contains(link, inode) {
			fields := strings.Split(procFd, "/") // eg /proc/2256/fd/9
			exe, err := os.Readlink("/proc/" + fields[2] + "/exe")
			if err != nil {
				continue
			}
			return strings.Title(filepath.Base(exe))
		}

	}

	return "Unknown"
}

func (obj *hostNetServices) appendEnvSetup(services *[]common.ServiceInfoEntry) {
	for _, e := range config.GetHostAllowedIPs() {
		entry := common.ServiceInfoEntry{
			Name: e.Name,
			Type: serviceType,
			IPs:  []string{e.Subnet},
			Ports: common.Ports{
				TCP: []uint16{},
				UDP: []uint16{},
			},
		}
		*services = append(*services, entry)
	}

}

func (obj *hostNetServices) execute() {
	services := []common.ServiceInfoEntry{}

	// Do not parse locally running services. They can anyway be reached via created tunnels
	// Leving commented code for some time, as this part may need to be reviewed once again
	// obj.parseProcNetFile("/proc/net/tcp", &services)
	// obj.parseProcNetFile("/proc/net/udp", &services)

	// Not yet
	//	obj.parseProcNetFile("/proc/net/tcp6", &services)
	//	obj.parseProcNetFile("/proc/net/udp6", &services)

	obj.appendEnvSetup(&services)

	if !cmp.Equal(services, obj.msg.Data) {
		obj.msg.Data = services
		obj.msg.Now()
		raw, err := json.Marshal(obj.msg)
		if err != nil {
			logger.Error().Println(pkgName, "json marshal", err)
			return
		}
		logger.Message().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(raw)
	}
}
