package model

import (
	"net"
	"time"
)

type Query struct {
	Id            int
	Source        net.Addr
	CreatedAt     time.Time
	Origin        Machine
	OriginId      int
	Destination   Host
	DestinationId int
}
