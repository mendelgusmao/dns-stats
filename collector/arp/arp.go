package arp

import (
	"fmt"
	"io/ioutil"
	"log"
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
			return fmt.Errorf("arp.Scan: Parsing '%s': %v", parts[3], err)
		}

		addEntry(parts[0], parts[3])
	}

	ifaces, err := net.Interfaces()

	if err != nil {
		return fmt.Errorf("arp.Scan: Scanning own interfaces: %v", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()

		if err != nil {
			return fmt.Errorf("arp.Scan: Getting addresses of interface %s: %v", iface.Name, err)
		}

		for _, addr := range addrs {
			ip := strings.Split(addr.String(), "/")[0]

			if iface.HardwareAddr.String() == "" {
				continue
			}

			addEntry(ip, iface.HardwareAddr.String())
		}
	}

	return nil
}

func addEntry(ip, mac string) {
	if _, ok := table[ip]; !ok {
		log.Printf("arp.Scan: %s (%s) is a new entry\n", ip, mac)
	}

	table[ip] = mac
}

func ping(ip string) (string, error) {
	output, err := exec.Command("ping", "-c1", ip).CombinedOutput()
	return string(output), err
}

func FindByIP(ip string) (string, error) {
	mtx.Lock()
	defer mtx.Unlock()

	_, ok := table[ip]

	if !ok {
		if output, err := ping(ip); err != nil {
			return Zero, fmt.Errorf("arp.FindByIP: Pinging '%s': %v -- %v", ip, err, output)
		}
	}

	return table[ip], nil
}
