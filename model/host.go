package model

import "time"

type Host struct {
	Id        int
	Address   string
	CreatedAt time.Time
}

func init() {
	register(Host{})
}
