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
	"github.com/multiformats/go-multiaddr"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"os"
	"io/ioutil"
	"strconv"
	"bft/consensus"
	"bft/types"
	"sync"
	"bft/database"
	"bft/encoding"
	crypto2 "bft/crypto"
)

type NetManager struct {
	mutex sync.Mutex
	host       		host.Host
	ipAddress  		string
	listenPort 		int
	targets    		[]string
	connections 	map[string]*Connection
	keyPair			types.KeyPair
	address			string
	chainId 		types.Hash
	consensusManager *consensus.ConsensusManager
	synchonizer		*Synchronizer
	dispatcher		*Dispatcher
}

func NewNetManager(ipAddress string, listenPort int, targets []string) *NetManager {
	netManager := &NetManager{
		ipAddress:		ipAddress,
		listenPort:		listenPort,
		targets:		targets,
		connections:	make(map[string]*Connection),
		chainId: 		database.GetBlockStore().ChainId(),
		synchonizer:	NewSynchronizer(),
	}
	////TODO: get initial validators
	//validators := types.Validators{}
	//enDecoder := types.EnDecoder{
	//	encoding.MarshalBinary,
	//	encoding.UnmarshalBinary,
	//}
	////TODO: load key pair from wallet
	privateKey, _ := crypto2.NewRandomPrivateKey()
	netManager.keyPair.PrivateKey = *privateKey
	netManager.keyPair.PublicKey = *privateKey.PublicKey()
	//signer := netManager.keyPair.PrivateKey.Sign
	//address := netManager.keyPair.PublicKey.Address()
	//netManager.consensusManager = consensus.NewConsensusManager(validators, address)
	//netManager.consensusManager.SetEnDecoder(enDecoder)
	//netManager.consensusManager.SetSigner(signer)
	//netManager.consensusManager.SetBroadcaster(netManager.broadcast)
	priv, err := loadIdentity(types.HostIdentity + strconv.Itoa(listenPort))
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
	nm.host.SetStreamHandler(pid, nm.handleInStream)
}

func (nm *NetManager) addPeers(targets []string) {
	for _, addr := range nm.targets {
		nm.addPeer(addr)
	}
}

func (nm *NetManager) handleInStream(s net.Stream) {
	conn := newConnection(s, nm.onReceive, nm.removeConnection)
	log.Printf("connected to inbound %s\n", conn.RemotePeerId())
	nm.addConnection(conn)
	conn.Start()
}

func (nm *NetManager) handleOutStream(s net.Stream) {
	conn := newConnection(s, nm.onReceive, nm.removeConnection)
	log.Printf("connected to outbound %s\n", conn.RemotePeerId())
	nm.addConnection(conn)
	nm.sendHandshake(conn)
	conn.Start()
}

func (nm *NetManager) onReceive(message types.Message, connection *Connection) {
	messageType := message.Type
	switch messageType {
	case types.HandshakeMessage:
		handshake := message.ToHandshake(encoding.UnmarshalBinary)
		nm.handleHandshake(handshake, connection)
	case types.VoteMessage, types.ProposalMessage:
		nm.consensusManager.Receive(message)
	}
}

func (nm *NetManager) broadcast(message types.Message) {
	for _, c := range nm.connections {
		c.Send(message)
	}
}

func (nm *NetManager) addPeer(peerAddress string) {
	fullAddr, err := multiaddr.NewMultiaddr(peerAddress)
	if err != nil {
		log.Println(err)
		return
	}
	pid, err := fullAddr.ValueForProtocol(multiaddr.P_IPFS)
	if err != nil {
		log.Println(err)
		return
	}
	peerId, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Println(err)
		return
	}
	ipfsPart, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerId)))
	targetAddr := fullAddr.Decapsulate(ipfsPart)
	nm.host.Peerstore().AddAddr(peerId, targetAddr, peerstore.PermanentAddrTTL)
	log.Println("opening stream")
	protocolId := protocol.ID(types.P2P + types.NetworkVersion)
	stream, err := nm.host.NewStream(context.Background(), peerId, protocolId)
	if err != nil {
		log.Println(err)
		return
	}
	nm.handleOutStream(stream)
}

func (nm *NetManager) addConnection(c *Connection) {
	nm.mutex.Lock()
	nm.connections[c.RemotePeerId()] = c
	nm.mutex.Unlock()
}

func (nm *NetManager) removeConnection(c *Connection) {
	nm.mutex.Lock()
	log.Println("disconnected peer from address ", c.RemotePeerId())
	c.Close()
	delete(nm.connections, c.RemotePeerId())
	nm.mutex.Unlock()
}

func (nm *NetManager) sendHandshake(c *Connection) {
	blockStore := database.GetBlockStore()
	lastHeightId := blockStore.Head().Header().HeightId
	signer := nm.keyPair.PrivateKey.Sign
	encoder := encoding.MarshalBinary
	handshake := types.NewHandshake(nm.chainId, nm.address, lastHeightId, signer, encoder)
	if handshake == nil {
		return
	}
	payload, err := encoding.MarshalBinary(*handshake)
	if err != nil {
		log.Println(err)
		return
	}
	message := types.NewMessage(types.HandshakeMessage, payload)
	c.Send(message)
	log.Println("sent handshake")
}

func (nm *NetManager) handleHandshake(handshake *types.Handshake, connection *Connection) {
	log.Println("received handshake")
	if handshake == nil {
		log.Println("unable to parse handshake")
		return
	}
	if !handshake.IsValid() {
		log.Println("handshake message is invalid")
		return
	}
	if !nm.chainId.Equals(handshake.ChainId) {
		log.Println("local chain id and remote chain id are not the same")
		return
	}
	if handshake.NetworkVersion != types.NetworkVersion {
		log.Println("network version does not match")
		return
	}
	if !handshake.Verify() {
		log.Println("handshake is not verified")
		return
	}
	if connection.lastSentHandshake == nil {
		log.Println("should send handshake")
		nm.sendHandshake(connection)
	}
	connection.lastReceivedHandshake = handshake
	//nm.synchonizer.handleHandshake(handshake, connection)
}

func loadIdentity(fileName string) (crypto.PrivKey, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return generateNewIdentity(fileName)
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	return crypto.UnmarshalPrivateKey(b)
}

func generateNewIdentity(fileName string) (crypto.PrivKey, error) {
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}
	b, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(fileName, b, 0644)
	if err != nil {
		return nil, err
	}
	return priv, nil
}