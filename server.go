package rpc

import (
    "context"
    "github.com/DGHeroin/rpc/pb"
    "net"
    "sync"
    "time"
)

type Server struct {
    ReadWriteTimeout time.Duration
    servant          *Servant
    onOpen           func(invokable Callable)
    onClose          func(invokable Callable)
    waitGroup        sync.WaitGroup
}

func (s *Server) ListenAndServe(addr string) error {
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    for {
        conn, err := ln.Accept()
        if err != nil {
            break
        }
        s.handleConn(conn)
    }
    return nil
}

type acceptClient struct {
    conn *Conn
    mgr  *CallManager
}

func (c *acceptClient) Call(ctx context.Context, service string, args interface{}, reply interface{}) error {
    call := c.mgr.newCall(c.conn)
    defer func() {
        c.mgr.remCall(call)
        close(call.done)
    }()
    return call.Call(ctx, service, args, reply)
}
func (s *Server) handleConn(conn net.Conn) {
    c := NewConn(conn, &s.waitGroup)
    c.OnMessage = func(msgType byte, msg *pb.Message) {
        switch msgType {
        case 1:
            if reply, err := s.servant.handleRequest(msg); err == nil {
                _ = c.Send(2, reply)
            }
        }
    }

    cli := &acceptClient{
        conn: c,
        mgr:  newCallManager(),
    }
    c.OnClose = func(conn *Conn) {
        s.onClose(cli)
    }
    s.onOpen(cli)
    c.Do()
}

func (s *Server) OnOpen(fn func(caller Callable)) {
    s.onOpen = fn
}

func (s *Server) OnClose(fn func(caller Callable)) {
    s.onClose = fn
}

func (s *Server) Register(serviceName string, i interface{}) bool {
    return s.servant.Register(serviceName, i)
}
