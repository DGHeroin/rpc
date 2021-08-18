package rpc

import (
    "context"
    "encoding/binary"
    "github.com/DGHeroin/rpc/pb"
    "log"
    "net"
    "sync"
    "time"
)

type Client struct {
    ReadWriteTimeout time.Duration
    servant          *Servant
    addr             string
    conn             *Conn
    mgr              *CallManager
    waitGroup        sync.WaitGroup
}

func (client *Client) Call(ctx context.Context, service string, args interface{}, reply interface{}) error {
    conn, err := client.GetConn()
    if err != nil {
        return err
    }
    call := client.mgr.newCall(conn)
    defer func() {
        client.mgr.remCall(call)
        close(call.done)
    }()
    err = call.Call(ctx, service, args, reply)
    return err
}

func (client *Client) GetConn() (*Conn, error) {
    if client.conn != nil {
        return client.conn, nil
    }
    conn, err := net.Dial("tcp", client.addr)
    if err != nil {
        return nil, err
    }
    client.conn = NewConn(conn, &client.waitGroup)
    client.conn.OnMessage = client.handleMessage
    client.conn.Do()
    return client.conn, nil
}

func (client *Client) handleMessage(msgType byte, msg *pb.Message) {
    switch msgType {
    case 0: // 心跳
    case 1: // request
        client.servant.handleRequest(msg)
    case 2: // response
        values := parseMessageValue(msg)
        idBin := values["Id"]
        id := binary.BigEndian.Uint32(idBin)
        call := client.mgr.popCall(id)
        if call == nil {
            log.Println("call 不存在")
            return
        }
        call.done <- values
    }

}

func (client *Client) Register(serviceName string, i interface{}) bool {
    return client.servant.Register(serviceName, i)
}
