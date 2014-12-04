package arp

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

const (
	localhost  = "127.0.0.1"
	arpContent = `IP address       HW type     Flags       HW address            Mask     Device
192.168.254.1     0x1         0x2         00:01:02:03:04:05     *        eth0
`
	arpContent2 = `IP address       HW type     Flags       HW address            Mask     Device
192.168.254.2     0x1         0x2         invalid-mac-address     *        eth0
`
)

type buffer struct {
	content [][]byte
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.content = append(b.content, (p))
	return len(b.content), nil
}

func TestAddEntry(t *testing.T) {
	table = make(map[string]string)

	b := &buffer{}
	log.SetOutput(b)

	addEntry(localhost, Zero)

	if len(b.content) == 0 {
		t.Error()
	}

	content := string(b.content[0])

	if !strings.Contains(content, fmt.Sprintf("%s (%s)", localhost, Zero)) {
		t.Fatal()
	}
}

func TestPing(t *testing.T) {
	content, err := ping("127.0.0.1")
	t.Log(content, err)
}

func TestFindByIP(t *testing.T) {
	table = make(map[string]string)

	addEntry(localhost, Zero)

	if _, err := FindByIP(localhost); err != nil {
		t.Fatal(err)
	}

	if _, err := FindByIP("192.168.254.254"); err == nil {
		t.Fatal(err)
	}
}

func TestScan(t *testing.T) {
	table = make(map[string]string)
	procNetARP = "/proc/net/invalid-arp"

	err := Scan()

	if _, ok := err.(*os.PathError); !ok {
		t.Fatal()
	}

	file, err := ioutil.TempFile("/tmp", "dns-stats-test")

	if err != nil {
		t.Error(err)
	}

	if _, err := io.WriteString(file, arpContent); err != nil {
		t.Error(err)
	}

	procNetARP = file.Name()

	err = Scan()

	if err != nil {
		t.Fatal(err)
	}

	file, err = ioutil.TempFile("/tmp", "dns-stats-test")

	if err != nil {
		t.Error(err)
	}

	if _, err := io.WriteString(file, arpContent2); err != nil {
		t.Error(err)
	}

	procNetARP = file.Name()

	err = Scan()

	if !strings.Contains(err.Error(), "Parsing") {
		t.Error()
	}
}
