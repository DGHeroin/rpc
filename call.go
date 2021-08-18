package rpc

import (
    "context"
    "fmt"
    "sync"
)

type (
    Callable interface {
        Call(ctx context.Context, service string, args interface{}, reply interface{}) error
    }
    Call struct {
        Id   uint32
        done chan map[string][]byte
        conn *Conn
    }
    CallManager struct {
        mutex  sync.Mutex
        reqMap map[uint32]*Call
        reqId  uint32
    }
)

func newCallManager() *CallManager {
    return &CallManager{
        reqMap: make(map[uint32]*Call),
        reqId:  0,
    }
}
func (call *Call) Call(ctx context.Context, service string, args interface{}, reply interface{}) error {
    req, err := buildRequest(call.Id, service, args)
    if err != nil {
        return err
    }
    err = call.conn.Send(1, req)
    if err != nil {
        return err
    }
    values := <-call.done
    if values["error"] != nil {
        return fmt.Errorf("%s", values["error"])
    }
    payload := values["payload"]
    err = Unmarshal(payload, reply)
    return err
}

func asyncDo(fn func(), wg *sync.WaitGroup) {
    wg.Add(1)
    go func() {
        fn()
        wg.Done()
    }()
}

func (m *CallManager) newCall(conn *Conn) *Call {
    call := &Call{
        Id:   m.nexId(),
        conn: conn,
        done: make(chan map[string][]byte),
    }
    m.addCall(call)
    return call
}
func (m *CallManager) nexId() uint32 {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    var reqId uint32
    for {
        if _, ok := m.reqMap[m.reqId]; !ok {
            reqId = m.reqId
            m.reqId++
            break
        }
    }
    return reqId
}
func (m *CallManager) addCall(call *Call) {
    m.mutex.Lock()
    m.reqMap[call.Id] = call
    m.mutex.Unlock()
}
func (m *CallManager) remCall(call *Call) {
    m.mutex.Lock()
    delete(m.reqMap, call.Id)
    m.mutex.Unlock()
}
func (m *CallManager) popCall(id uint32) *Call {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    return m.reqMap[id]
}
