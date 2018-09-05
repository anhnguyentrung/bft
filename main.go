package main

import (
	"bft/network"
)

func main() {
	//targets := []string{"/ip4/127.0.0.1/tcp/2000/ipfs/QmVJih4nhy6TGVrJKBmtinPFimDD83N3FrmMyQPZmNF7hE"}
	targets := make([]string, 0)
	nm := network.NewNetManager("127.0.0.1", 2000, targets)
	//done := make(chan struct{})
	nm.Run()
	//<- done
}
