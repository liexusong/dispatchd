
package main

import (
  "fmt"
  "net"
  // "errors"
  // "os"
  "bytes"
  "github.com/jeffjenkins/mq/amqp"
)

type ConnectStatus struct {
  start bool
  startOk bool
  secure bool
  secureOk bool
  tune bool
  tuneOk bool
  open bool
  openOk bool
}

type AMQPConnection struct {
  network net.Conn
  open bool
  done bool
  // this is thread-safe because only channel 0 manages these fields
  nextChannel int
  channels map[uint16]*Channel
  outgoing chan *amqp.FrameWrapper
  connectStatus ConnectStatus
}

func NewAMQPConnection(network net.Conn) *AMQPConnection {
  return &AMQPConnection{
    network,
    false,
    false,
    0,
    make(map[uint16]*Channel),
    make(chan *amqp.FrameWrapper),
    ConnectStatus{},
  }
}

func (conn *AMQPConnection) openConnection() {
    fmt.Println("=====> Incoming connection!")
  // Negotiate Protocol
  buf := make([]byte, 8)
  _, err := conn.network.Read(buf)
  if err != nil {
    fmt.Println("Error reading:", err.Error())
  }

  var supported = []byte { 'A', 'M', 'Q', 'P', 0, 0, 9, 1 }
  if bytes.Compare(buf, supported) != 0 {
    conn.network.Write(supported)
    conn.network.Close()
    fmt.Println("Bad version from client")
    return
  }

  // Create channel 0 and start the connection handshake
  conn.channels[0] = NewChannel(0, conn)
  conn.channels[0].start()
  conn.handleOutgoing()
  conn.handleIncoming()
}

func (conn *AMQPConnection) cleanUp() {

}

func (conn *AMQPConnection) handleOutgoing() {
  go func() {
    for {
      var frame = <- conn.outgoing
      amqp.WriteFrame(conn.network, frame)
    }
  }()
}

func (conn *AMQPConnection) handleIncoming() {
  for {
    // If the connection is done, we stop handling frames
    if conn.done {
      break
    }
    // Read from the network
    frame, err := amqp.ReadFrame(conn.network)
    if err != nil {
      fmt.Println("Error reading frame")
      conn.network.Close()
      break
    }
    fmt.Println("Got frame from client")
    // Cleanup. Remove things which have expired, etc
    conn.cleanUp()

    // TODO: handle non-method frames (maybe?)

    // If we haven't finished the handshake, ignore frames on channels other
    // than 0
    if !conn.open && frame.Channel != 0 {
      fmt.Println("Non-0 channel for unopened connection")
      conn.network.Close()
      break
    }
    var channel, ok = conn.channels[frame.Channel]
    if !ok {
      channel = NewChannel(frame.Channel, conn)
      conn.channels[frame.Channel] = channel
      conn.channels[frame.Channel].start()
    }
    // Dispatch
    fmt.Println("Sending frame to", channel.id)
    channel.incoming <- frame
  }
}