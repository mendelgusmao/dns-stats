package report

import (
	"strconv"
	"strings"
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

func (v vector) value(in string) (out int) {
	blocks := strings.Split(in, ".")
	block := blocks[len(blocks)-1]
	out, _ = strconv.Atoi(block)
	return
}
