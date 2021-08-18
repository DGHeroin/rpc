package main

import (
    "context"
    "github.com/DGHeroin/rpc"
    "github.com/DGHeroin/rpc/example/shard"
    "log"
)

func main()  {
    server := rpc.NewP2PServer()
    server.OnOpen(func(caller rpc.Callable) {
        log.Println("新连接", caller)
    })
    server.OnClose(func(caller rpc.Callable) {
        log.Println("关闭", caller)
    })
    server.Register("service.hello", onMessage)

    errCh := make(chan error)
    server.ListenAndServe(":1600")
    for {
        select {
        case err := <- errCh:
            log.Println(err)
        }
    }
}
func onMessage(ctx context.Context, request *shard.SimpleRequest, reply *shard.SimpleReply) error {
 //   log.Println("onMessage", request, reply)

    reply.Message = "I got it"
    return nil
}