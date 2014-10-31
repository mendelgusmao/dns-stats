package model

import "time"

type Machine struct {
	Id        int
	IP        int
	MAC       string
	CreatedAt time.Time
}
