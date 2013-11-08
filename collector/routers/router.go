package routers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	routers   = make(map[string]*regexp.Regexp)
	noMatches = errors.New("Couldn't extract data from message")
)

type Router interface {
	name() string
	message() string
}

func register(router Router) {
	expression := regexp.MustCompile(router.message())
	captures := 0

	for _, name := range expression.SubexpNames() {
		if name == "origin" || name == "destination" {
			captures++
		}
	}

	if captures != 2 {
		fmt.Printf("Router %s is not going to be registered: absence or excess of named captures (origin, destination)\n", router.name())
		return
	}

	fmt.Printf("Registering router %s\n", router.name())
	routers[router.name()] = expression
}

func Registered() string {
	registered := make([]string, 0)

	for name, _ := range routers {
		registered = append(registered, name)
	}

	return strings.Join(registered, ", ")
}

func Find(name string) *regexp.Regexp {
	expression, ok := routers[name]

	if !ok {
		return nil
	}

	return expression
}

func Extract(expression *regexp.Regexp, matches []string) (origin, destination string, err error) {
	if len(matches) < 2 {
		err = noMatches
		return
	}

	for index, name := range expression.SubexpNames() {
		if name == "origin" {
			origin = matches[index]
		}

		if name == "destination" {
			destination = matches[index]
		}
	}

	return
}
