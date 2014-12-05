package routers

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/MendelGusmao/dns-stats/model"
)

var (
	routers = make(map[string]*regexp.Regexp)
)

func Register(routerName, message string) {
	re := regexp.MustCompile(message)
	captures := make(map[string]bool)

	for _, name := range re.SubexpNames() {
		if name == "origin" || name == "destination" {
			captures[name] = true
		}
	}

	if len(captures) != 2 {
		log.Printf("routers.Register: %s is not going to be registered: absence or excess of named captures (origin, destination)\n", routerName)
		return
	}

	log.Printf("routers.Register: registered %s\n", routerName)
	routers[routerName] = re
}

func List() string {
	registered := make([]string, 0)

	for name, _ := range routers {
		registered = append(registered, name)
	}

	return strings.Join(registered, ", ")
}

func Find(name string) *regexp.Regexp {
	re, ok := routers[name]

	if !ok {
		return nil
	}

	return re
}

func Extract(expression *regexp.Regexp, content string) (*model.Query, error) {
	matches := expression.FindStringSubmatch(content)

	if len(matches) < 2 {
		return nil, fmt.Errorf("Couldn't extract data from message (%s)", content)
	}

	query := &model.Query{}

	for index, name := range expression.SubexpNames() {
		if name == "origin" {
			query.Origin = model.Machine{}.SetIP(matches[index])
		}

		if name == "destination" {
			query.Destination = model.Host{Address: matches[index]}
		}
	}

	return query, nil
}
