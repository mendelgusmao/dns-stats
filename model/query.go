package model

import (
	"net"
	"time"
)

type Query struct {
	Id            int
	Source        net.Addr
	Origin        Machine
	OriginId      int
	Destination   Host
	DestinationId int
	At            time.Time
}
