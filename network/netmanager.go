package network

import (
	"github.com/libp2p/go-libp2p-host"
	"crypto/rand"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p"
	"fmt"
	"context"
	"github.com/libp2p/go-libp2p-net"
	"log"
	"bufio"
)

type NetManager struct {
	host host.Host
	ipAddress string
	listenPort int
}

func NewNetManager(ipAddress string, listenPort int) *NetManager {
	netManager := &NetManager{
		ipAddress:ipAddress,
		listenPort:listenPort,
	}
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("ip4/%s/tcp/%d", ipAddress, listenPort)),
		libp2p.Identity(priv),
	}
	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil
	}
	netManager.host = host
	return netManager
}

func (nm *NetManager) handleStream(s net.Stream) {
	log.Println("Got a new stream")
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

}

func (nm *NetManager) readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if str == "" {
			return
		}
		if str != "\n" {

		}
	}
}

func (nm *NetManager) writeData() {

}