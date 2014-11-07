package report

import (
	"encoding/binary"
	"net"
)

type vector []string

func (v vector) Len() int {
	return len(v)
}

func (v vector) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v vector) Less(i, j int) bool {
	return v.value(v[i]) > v.value(v[j])
}

func (v vector) value(in string) int {
	return int(binary.BigEndian.Uint32(net.ParseIP(in).To4()))
}
