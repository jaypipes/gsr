package util

import (
	"log"
	"net"
	"strings"
)

func BindHost() string {
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	addr := c.LocalAddr().String()
	return addr[:strings.LastIndex(addr, ":")]
}
