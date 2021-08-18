package rpc

import (
    "context"
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
    client.conn.OnClose = func(conn *Conn) {
        log.Println("关闭连接...")
        client.conn = nil
    }
    return client.conn, nil
}

func (client *Client) handleMessage(msgType byte, msg *pb.Message) {
    switch msgType {
    case 0: // 心跳
    case 1: // request
        reply := client.servant.handleRequest(msg)
        client.conn.Send(2, reply)
    case 2: // response
        call := client.mgr.popCall(*msg.Id)
        if call == nil {
            log.Println("call 不存在")
            return
        }
        call.done <- msg
    }

}

func (client *Client) Register(serviceName string, i interface{}) bool {
    return client.servant.Register(serviceName, i)
}
