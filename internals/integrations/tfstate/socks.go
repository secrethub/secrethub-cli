package tfstate

import (
	"net"
)

type Connection struct {
	LocalAddress  net.IP
	LocalPort     uint16
	RemoteAddress net.IP
	RemotePort    uint16
	Process
}

type Process struct {
	PID  int
	Name string
}
