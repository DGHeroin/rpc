package main

import (
    "context"
    "github.com/DGHeroin/rpc"
    "github.com/DGHeroin/rpc/example/shard"
    "log"
    "sync/atomic"
    "time"
)

var (
    count uint32
)

func sendRequest(client *rpc.Client) {
    var (
        req   shard.SimpleRequest
        reply shard.SimpleReply
    )
    req.ActionName = "I want to say hello"
    client.Call(context.Background(), "service.hello", &req, &reply)
    atomic.AddUint32(&count, 1)
    //log.Println(err, reply.Message)
}
func openOneClient()  {
    client := rpc.NewP2PClient("127.0.0.1:1600")
    for {
       sendRequest(client)
    }
}
func main() {

    go func() {
        lastVal := uint32(0)
        for {
            time.Sleep(time.Second)
            now := atomic.LoadUint32(&count)
            log.Println("count:", now, now - lastVal)
            lastVal = now
        }
    }()
    for i := 0; i < 1000; i++ {
        openOneClient()
    }
    select {

    }
    //for {
    //    sendRequest(client)
    //}

}
