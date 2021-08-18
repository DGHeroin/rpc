package rpc

import "time"

func NewP2PServer() *Server {
    return &Server{
        ReadWriteTimeout: time.Second * 10,
        servant:          NewServant(),
    }
}

func NewP2PClient(addr string) *Client {
    cli := &Client{
        ReadWriteTimeout: time.Second * 10,
        servant:          NewServant(),
        addr:             addr,
        mgr:              newCallManager(),
    }
    return cli
}
