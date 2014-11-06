package model

import (
	"encoding/binary"
	"net"
	"time"
)

type Machine struct {
	Id        int
	IP        uint32
	MAC       string
	CreatedAt time.Time
	StringIP  string `sql:"-"`
}

func (m Machine) SetIP(ip string) Machine {
	m.StringIP = ip
	m.IP = binary.LittleEndian.Uint32(net.ParseIP(ip).To4())
	return m
}

func init() {
	register(Machine{})
}
