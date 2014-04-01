package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"log/syslog"
	"math/rand"
	"strings"
	"time"
)

func main() {
	logger, err := syslog.NewLogger(syslog.LOG_SYSLOG|syslog.LOG_LOCAL0, log.LstdFlags)

	if err != nil {
		fmt.Println("error creating logger", err)
		return
	}

	logger.Printf("dns-stats 200.160.3.254,%s,200.160.3.93", randstr())
}

func randstr() string {
	t := time.Now().UnixNano()
	rand.Seed(t)
	r := rand.Int63n(2 << 60)
	s := fmt.Sprintf("%d", r)
	u := base64.StdEncoding.EncodeToString([]byte(s))
	u = strings.ToLower(u)
	u = strings.Replace(u, "=", "", -1)
	return fmt.Sprintf("%s.com", u)
}
