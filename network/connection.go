package network

import (
	"bft/types"
	"bufio"
	"sync"
	"fmt"
	"log"
	"github.com/libp2p/go-libp2p-net"
	"bft/encoding"
)

type ReceiveFunc func (message types.Message)
type FinishFunc func (connection *Connection)

type Connection struct {
	mutex sync.Mutex
	stream net.Stream
	readWriter *bufio.ReadWriter
	lastHeightId types.BlockHeightId
	syncing bool
	onReceive ReceiveFunc
	onFinish FinishFunc
}

func newConnection(stream net.Stream, onReceive ReceiveFunc, onFinish FinishFunc) *Connection {
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	return &Connection{
		stream:stream,
		readWriter:rw,
		syncing:false,
		onReceive:onReceive,
		onFinish:onFinish,
	}
}

func (c *Connection) Send(message types.Message) error {
	buf, err := encoding.MarshalBinary(message)
	if err != nil {
		return err
	}
	c.mutex.Lock()
	c.readWriter.WriteString(fmt.Sprintf("%s\n", string(buf)))
	c.readWriter.Flush()
	c.mutex.Unlock()
	return err
}

func (c *Connection) Start() {
	go c.readLoop()
}

func (c *Connection) LocalPeerId() string {
	return c.stream.Conn().LocalPeer().String()
}

func (c *Connection) RemotePeerId() string {
	return c.stream.Conn().RemotePeer().String()
}

func (c *Connection) Close() {
	c.readWriter = nil
	c.syncing = false
	c.stream.Close()
}

func (c *Connection) Sync(syncing bool) {
	c.syncing = syncing
}

func (c *Connection) IsAvailable() bool {
	return c.readWriter != nil && !c.syncing
}

func (c *Connection) readLoop() {
	for {
		str, err := c.readWriter.ReadString('\n')
		if err != nil {
			c.onFinish(c)
			return
		}
		if str == "" {
			c.onFinish(c)
			return
		}
		if str != "\n" {
			message := types.Message{
				Header:		types.MessageHeader{},
				Payload: 	make([]byte, 0),
			}
			err = encoding.UnmarshalBinaryMessage([]byte(str), &message)
			if err != nil {
				log.Fatal(err)
			}
			c.onReceive(message)
		}
	}
}
