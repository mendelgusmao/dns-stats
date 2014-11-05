package arp

import (
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"
	"sync"
)

var (
	table = make(map[string]string)
	mtx   sync.RWMutex
)

const (
	procNetARP = "/proc/net/arp"
	Zero       = "00:00:00:00:00:00"
)

func Scan() error {
	content, err := ioutil.ReadFile(procNetARP)

	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	mtx.Lock()
	defer mtx.Unlock()

	for index, line := range lines {
		if index == 0 || len(line) == 0 {
			continue
		}

		for strings.Contains(line, "  ") {
			line = strings.Replace(line, "  ", " ", -1)
		}

		parts := strings.Split(line, " ")
		_, err := net.ParseMAC(parts[3])

		if err != nil {
			return fmt.Errorf("arp.go: Parsing '%s': %v", parts[3], err)
		}

		table[parts[0]] = parts[3]
	}

	return nil
}

func ping(ip string) (string, error) {
	output, err := exec.Command("ping", "-c1", ip).CombinedOutput()
	return string(output), err
}

func FindByIP(ip string) (string, error) {
	mtx.Lock()
	defer mtx.Unlock()

	hwAddr, ok := table[ip]

	if !ok {
		output, err := ping(ip)

		if err != nil {
			return "", fmt.Errorf("arp.go: Pinging '%s': %v -- %v", ip, err, output)
		}

		hwAddr = table[ip]
	}

	return hwAddr, nil
}
