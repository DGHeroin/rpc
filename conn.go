package rpc

import (
    "github.com/DGHeroin/rpc/pb"
    "google.golang.org/protobuf/proto"
    "net"
    "sync"
    "time"
)

type (
    recvPacket struct {
        header  [HeaderSize]byte
        payload *pb.Message
    }
    Conn struct {
        conn              net.Conn
        wg                *sync.WaitGroup
        closeCh           chan struct{}
        closeOnce         sync.Once
        mutex             sync.Mutex
        ReadTimeout       time.Duration
        OnMessage         func(msgType byte, msg *pb.Message)
        OnClose           func(conn *Conn)
        packetSendChan    chan *pb.Message
        packetReceiveChan chan *recvPacket
    }
)

func NewConn(conn net.Conn, wg *sync.WaitGroup) *Conn {
    call := &Conn{
        conn: conn,
        wg:   wg,
        closeCh:           make(chan struct{}),
        ReadTimeout:       time.Second * 10,
        packetSendChan:    make(chan *pb.Message),
        packetReceiveChan: make(chan *recvPacket),
    }
    return call
}
func (c *Conn) Do() {
    asyncDo(c.handleLoop, c.wg)
    asyncDo(c.readLoop, c.wg)
    asyncDo(c.writeLoop, c.wg)
}
func (c *Conn) handleLoop() {
    defer func() {
        recover()
        c.Close()
    }()
    for {
        select {
        case <-c.closeCh:
            return
        case msg := <-c.packetReceiveChan:
            msgType := headerTypeCode(msg.header)
            c.onMessage(msgType, msg.payload)
        default:
        }
    }
}
func (c *Conn) readLoop() {
    defer func() {
        recover()
        c.Close()
    }()
    for {
        select {
        case <-c.closeCh:
            return
        default:

        }
        if err := c.conn.SetReadDeadline(time.Now().Add(c.ReadTimeout)); err != nil {
            return
        }
        header, pkt, err := c.readPacket()
        if err != nil {
            return
        }
        c.packetReceiveChan <- &recvPacket{
            header:  header,
            payload: pkt,
        }
    }
}
func (c *Conn) writeLoop() {
    defer func() {
        recover()
        c.Close()
    }()
    for {
        select {
        case <-c.closeCh:
            return
        case msg := <-c.packetSendChan:
            if err := c.Send(0, msg); err != nil {
                return
            }
        }
    }
}

func (c *Conn) Close() {
    c.closeOnce.Do(func() {
        _ = c.conn.Close()
        close(c.closeCh)
        if c.OnClose != nil {
            c.OnClose(c)
        }
    })

}

func (c *Conn) readPacket() ([HeaderSize]byte, *pb.Message, error) {
    var (
        err     error
        header  [HeaderSize]byte
        payload []byte
    )
    conn := c.conn
    if header, err = readHeader(conn, c.ReadTimeout); err != nil {
        return header, nil, err
    }
    if !headerValidMagic(header) {
        return header, nil, ErrMagicCode
    }
    switch headerTypeCode(header) {
    case 0: // heart beat
        return header, nil, nil
    case 1: // request call
        payloadSize := headerGetPayloadSize(&header)
        if payload, err = readPayload(conn, payloadSize, c.ReadTimeout); err != nil {
            return header, nil, err
        }

        crcNum := headerCrc(header)
        if num, _ := CalcCrc(payload); num != crcNum {
            return header, nil, ErrHeaderCRC
        }
        msg := &pb.Message{}
        if err = proto.Unmarshal(payload, msg); err != nil {
            return header, nil, err
        }
        return header, msg, nil
    case 2:
        payloadSize := headerGetPayloadSize(&header)
        if payload, err = readPayload(conn, payloadSize, c.ReadTimeout); err != nil {
            return header, nil, err
        }

        crcNum := headerCrc(header)
        if num, _ := CalcCrc(payload); num != crcNum {
            return header, nil, ErrHeaderCRC
        }
        msg := &pb.Message{}
        if err = proto.Unmarshal(payload, msg); err != nil {
            return header, nil, err
        }
        return header, msg, nil
    }
    return header, nil, ErrHeaderType
}

func (c *Conn) Send(msgType byte, msg *pb.Message) error {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    bin, err := makePkt(msgType, msg)
    if err != nil {
        return err
    }
    _, err = c.conn.Write(bin)
    return err
}

func (c *Conn) onMessage(msgType byte, msg *pb.Message) {
    defer func() {
        recover()
    }()
    c.OnMessage(msgType, msg)
}
