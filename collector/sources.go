package collector

import (
	"dns-stats/collector/routers"
	"errors"
	"fmt"
	"strings"
)

type Source struct {
	Host   string
	Router string
}

type Sources []Source

func (s *Sources) String() string {
	return fmt.Sprint(*s)
}

func (s *Sources) Set(value string) error {
	for _, source := range strings.Split(value, ",") {
		pair := strings.Split(source, ":")

		if len(pair) != 2 {
			return errors.New(fmt.Sprintf("Unrecognized source '%s'. It should be in format <address>:<router-name>", source))
		}

		if router := routers.Find(pair[1]); router == nil {
			return errors.New(fmt.Sprintf("Router '%s' is not registered", pair[1]))
		}

		*s = append(*s, Source{
			Host:   pair[0],
			Router: pair[1],
		})
	}

	return nil
}
