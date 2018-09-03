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
	"bft/network/types"
	"github.com/multiformats/go-multiaddr"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
)

type NetManager struct {
	host 		host.Host
	ipAddress 	string
	listenPort 	int
	targets		[]string
}

func NewNetManager(ipAddress string, listenPort int, targets []string) *NetManager {
	netManager := &NetManager{
		ipAddress:ipAddress,
		listenPort:listenPort,
		targets:targets,
	}
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddress, listenPort)),
		libp2p.Identity(priv),
	}
	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil
	}
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	log.Printf("address: %s", fullAddr)
	netManager.host = host
	return netManager
}

func (nm *NetManager) Run() {
	nm.listen()
	nm.addPeers(nm.targets)
	select {}
}

func (nm *NetManager) listen() {
	pid := protocol.ID(types.P2P + types.NetworkVersion)
	nm.host.SetStreamHandler(pid, nm.handleStream)
}

func (nm *NetManager) addPeers(targets []string) {
	for _, addr := range nm.targets {
		nm.addPeer(addr)
	}
}

func (nm *NetManager) handleStream(s net.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go nm.readData(rw)
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
			message := types.Message{
				Header:		types.MessageHeader{},
				Payload: 	make([]byte, 0),
			}
			err = UnmarshalBinaryMessage([]byte(str), &message)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (nm *NetManager) writeData() {

}

func (nm *NetManager) addPeer(peerAddress string) {
	fullAddr, err := multiaddr.NewMultiaddr(peerAddress)
	if err != nil {
		log.Fatal(err)
	}
	pid, err := fullAddr.ValueForProtocol(multiaddr.P_IPFS)
	if err != nil {
		log.Fatal(err)
	}
	peerId, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatal(err)
	}
	ipfsPart, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerId)))
	targetAddr := fullAddr.Decapsulate(ipfsPart)
	nm.host.Peerstore().AddAddr(peerId, targetAddr, peerstore.PermanentAddrTTL)
	log.Println("opening stream")
	protocolId := protocol.ID(types.P2P + types.NetworkVersion)
	stream, err := nm.host.NewStream(context.Background(), peerId, protocolId)
	if err != nil {
		log.Fatal(err)
	}
	nm.handleStream(stream)
}