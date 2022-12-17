package main

import (
	"fmt"
	"net"
	"net/netip"
	"time"
)

func toNetAddr(addr netip.Addr) net.Addr {
	return &net.IPAddr{IP: addr.AsSlice()}
}

func to4(addr netip.Addr) []byte {
	b := addr.As4()
	return b[:]
}

func mustAddrFromSlice(b []byte) netip.Addr {
	addr, ok := netip.AddrFromSlice(b)
	if !ok {
		panic("mustAddrFromSlice: slice should be either 4 or 16 bytes, but got " + fmt.Sprint(len(b)))
	}
	return addr
}

func tickImmediately(d time.Duration) <-chan time.Time {
	c := make(chan time.Time)

	go func() {
		c <- time.Now()
		for t := range time.Tick(d) {
			c <- t
		}
	}()

	return c
}
