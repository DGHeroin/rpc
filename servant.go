package rpc

import (
    "bytes"
    "context"
    "fmt"
    "github.com/DGHeroin/rpc/pb"
    "google.golang.org/protobuf/proto"
    "log"
    "reflect"
    "runtime"
)

type (
    Servant struct {
        handler map[string]*ServantHandle
    }
    ServantHandle struct {
        fn reflect.Value
        r  reflect.Type
        w  reflect.Type
    }
)

func NewServant() *Servant {
    return &Servant{
        handler: make(map[string]*ServantHandle),
    }
}

func (s *Servant) Register(serviceName string, i interface{}) bool {
    sh, ok := checkFunc(i)
    if !ok {
        log.Println("注册失败", serviceName)
        return false
    }
    s.handler[serviceName] = sh
    return true
}

func (s *Servant) handleFunc(req *pb.Message) (reply *pb.Message) {
    reply = &pb.Message{}

    defer func() {
        if e := recover(); e != nil {
            buffer := bytes.NewBufferString(fmt.Sprint(e))
            //打印调用栈信息
            buf := make([]byte, 2048)
            n := runtime.Stack(buf, false)
            stackInfo := fmt.Sprintf("\n%s", buf[:n])
            buffer.WriteString(fmt.Sprintf("panic stack info %s", stackInfo))
            log.Println(buffer)
            //replyValues["error"] = []byte(fmt.Sprint(e))
        }
    }()
    sh, ok := s.handler[*req.Name]
    if !ok {
        log.Println("找不到函数")
        reply.Error = proto.String(ErrHandleNotFound.Error())
        return
    }

    ctx := context.Background()
    t0 := reflect.ValueOf(ctx)
    t1 := reflect.New(sh.r)
    t2 := reflect.New(sh.w)
    in := []reflect.Value{
        t0, t1, t2,
    }

    reply.Id = proto.Uint32(*req.Id)
    err := Unmarshal(req.Payload, t1.Interface())
    if err != nil {
        reply.Error = proto.String(ErrHandleNotFound.Error())
        return
    }

    rs := sh.fn.Call(in)
    r1 := rs[0]
    if r1.Interface() != nil {
        reply.Error = proto.String(fmt.Sprint(r1.Interface()))
        return
    }
    data, err := Marshal(in[2].Interface())
    if err != nil {
        reply.Error = proto.String(fmt.Sprint(r1.Interface()))
        return
    }

    reply.Payload = data
    return
}

func (s *Servant) handleRequest(msg *pb.Message) *pb.Message {
    reply := s.handleFunc(msg)

    reply.Action = proto.Int32(2)
    return reply
}
