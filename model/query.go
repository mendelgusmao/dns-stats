package model

import (
	"net"
	"time"
)

type Query struct {
	Id            int
	Source        string
	Origin        Machine
	OriginId      int
	Destination   Host
	DestinationId int
	At            time.Time
}

func (q Query) SetSource(addr net.Addr) {
	q.Source = addr.String()
}

func init() {
	register(&Query{})
}
