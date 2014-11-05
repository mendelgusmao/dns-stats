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
}

func (m Machine) SetIP(ip string) Machine {
	m.IP = binary.LittleEndian.Uint32(net.ParseIP(ip).To4())
	return m
}
