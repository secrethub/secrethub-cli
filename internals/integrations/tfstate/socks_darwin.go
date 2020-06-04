// +build darwin

package tfstate

import (
	"github.com/john-pierce/procspy"
)

func tcpSocks() ([]Connection, error) {
	cs, err := procspy.Connections(true)
	if err != nil {
		return nil, err
	}
	var res []Connection

	for sock := cs.Next(); sock != nil; sock = cs.Next() {
		res = append(res, Connection{
			LocalAddress:  sock.LocalAddress,
			LocalPort:     sock.LocalPort,
			RemoteAddress: sock.RemoteAddress,
			RemotePort:    sock.RemotePort,
			Process: Process{
				PID:  int(sock.Proc.PID),
				Name: sock.Proc.Name,
			},
		})
	}
	return res, nil
}
