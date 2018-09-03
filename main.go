package main

import (
	"bft/network"
)

func main() {
	targets := []string{"/ip4/127.0.0.1/tcp/2000/ipfs/QmXXj33UusZuhM2K6GtC93wfWiBsqwmrJg2CPvh14z3Et4"}
	//targets := make([]string, 0)
	nm := network.NewNetManager("127.0.0.1", 2001, targets)
	//done := make(chan struct{})
	nm.Run()
	//<- done
}
