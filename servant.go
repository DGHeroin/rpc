package rpc

import (
    "bytes"
    "fmt"
    "github.com/DGHeroin/rpc/pb"
    "google.golang.org/protobuf/proto"
    "log"
    "runtime"
)

type Servant struct {
    handler map[string]interface{}
}

func NewServant() *Servant {
    return &Servant{
        handler: make(map[string]interface{}),
    }
}

func (s *Servant) Register(serviceName string, i interface{}) bool {
    _, _, ok := checkFunc(i)
    if !ok {
        return false
    }
    s.handler[serviceName] = i
    return true
}

func (s *Servant) handleFunc(fn interface{}, req *pb.Message) (reply *pb.Message) {
    reply = &pb.Message{}
    var err error
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
    in, f, ok := checkFunc(fn)
    if !ok {
        return
    }

    reply.Id = proto.Uint32(*req.Id)

    //
    err = Unmarshal(req.Payload, in[1].Interface())
    if err != nil {
        log.Println(err)
        return
    }

    rs := f.Call(in)
    r1 := rs[0]
    if r1.Interface() != nil {
      //  replyValues["error"] = []byte(fmt.Sprint(r1))
    }
    data, err := Marshal(in[2].Interface())
    if err != nil {
        log.Println(err)
        return
    }

    reply.Payload = data
    return
}

func (s *Servant) handleRequest(msg *pb.Message) (*pb.Message, error) {
    handler, ok := s.handler[*msg.Name]
    if !ok {
        return nil, ErrHandleNotFound
    }
    reply := s.handleFunc(handler, msg)
    reply.Action = proto.Int32(2)
    return reply, nil
}
