// +build !darwin
// +build !windows !386

package tfstate

import (
	"github.com/cakturk/go-netstat/netstat"
)

func tcpSocks() ([]Connection, error) {
	socks, err := netstat.TCPSocks(netstat.NoopFilter)
	if err != nil {
		return nil, err
	}

	res := make([]Connection, len(socks))

	for i, sock := range socks {
		if sock.Process == nil {
			continue
		}
		res[i] = Connection{
			LocalAddress:  sock.LocalAddr.IP,
			LocalPort:     sock.LocalAddr.Port,
			RemoteAddress: sock.RemoteAddr.IP,
			RemotePort:    sock.RemoteAddr.Port,
			Process: Process{
				PID:  sock.Process.Pid,
				Name: sock.Process.Name,
			},
		}
	}
	return res, nil
}
