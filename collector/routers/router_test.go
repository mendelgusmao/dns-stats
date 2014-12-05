package routers

import (
	"log"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/MendelGusmao/dns-stats/model"
)

const (
	validExpression   = "(?P<origin>.*)--(?P<destination>.*)"
	invalidExpression = ".*"
)

type buffer struct {
	content [][]byte
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.content = append(b.content, (p))
	return len(b.content), nil
}

func (b *buffer) clear() {
	b.content = make([][]byte, 0)
}

func TestRegister(t *testing.T) {
	routers = make(map[string]*regexp.Regexp)

	b := &buffer{}
	log.SetOutput(b)

	Register("test", invalidExpression)

	if len(b.content) == 0 {
		t.Error()
	}

	content := string(b.content[0])

	if !strings.Contains(content, "test is not going to be registered") {
		t.Fatal()
	}

	b.clear()

	Register("test", validExpression)

	content = string(b.content[0])

	if !strings.Contains(content, "registered test") {
		t.Fatal()
	}
}

func TestList(t *testing.T) {
	routers = make(map[string]*regexp.Regexp)

	Register("test", validExpression)
	Register("test1", validExpression)

	if List() != "test, test1" {
		t.Fatal()
	}
}

func TestFind(t *testing.T) {
	routers = make(map[string]*regexp.Regexp)

	Register("test", "(?P<origin>.*) (?P<destination>.*)")

	if Find("test") == nil {
		t.Fatal()
	}

	if Find("test1") != nil {
		t.Fatal()
	}
}

func TestExtract(t *testing.T) {
	Register("test", validExpression)
	_, err := Extract(Find("test"), "invalid message")

	if _, ok := err.(error); !ok {
		t.Fatal(err)
	}

	query, err := Extract(Find("test"), "192.168.0.1--exemplo.br")

	expected := model.Query{
		Origin:      model.Machine{}.SetIP("192.168.0.1"),
		Destination: model.Host{Address: "exemplo.br"},
	}

	if !reflect.DeepEqual(query, &expected) {
		t.Fatal()
	}
}
