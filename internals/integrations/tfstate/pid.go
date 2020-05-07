package tfstate

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
)

func connectionFromChildProcess(pid int, r *http.Request) (bool, error) {
	split := strings.Split(r.RemoteAddr, ":")
	host := net.ParseIP(split[0])
	port64, err := strconv.ParseUint(split[1], 10, 16)
	if err != nil {
		return false, err
	}
	port := uint16(port64)

	socks, err := tcpSocks()
	if err != nil {
		return false, err
	}
	for _, c := range socks {
		if c.LocalAddress.Equal(host) && c.LocalPort == port {
			nextProcess := c.Process.PID
			for {
				if nextProcess == pid {
					return true, nil
				}
				if nextProcess == 1 {
					break
				}
				parent, err := ps.FindProcess(nextProcess)
				if err != nil {
					return false, err
				}
				if parent == nil {
					break
				}

				nextProcess = parent.PPid()
			}
		}
	}
	return false, nil
}
