package rpc

import (
    "bytes"
    "context"
    "encoding/binary"
    "fmt"
    "github.com/DGHeroin/rpc/pb"
    "google.golang.org/protobuf/proto"
    "hash/crc32"
    "io"
    "log"
    "net"
    "reflect"
    "runtime"
    "time"
)

const (
    HeaderSize = 14
)

var (
    ErrMagicCode      = fmt.Errorf("data magic error")
    ErrHeaderCRC      = fmt.Errorf("header crc error")
    ErrPayloadSize    = fmt.Errorf("data size error")
    ErrHeaderType     = fmt.Errorf("unknow header type")
    ErrHandleNotFound = fmt.Errorf("handler not found")
    ErrBadData        = fmt.Errorf("bad data")
)

func readHeader(conn net.Conn, duration time.Duration) ([HeaderSize]byte, error) {
    var (
        n      int
        err    error
        header = [HeaderSize]byte{}
    )
    if duration > 0 {
        err = conn.SetDeadline(time.Now().Add(duration))
        if err != nil {
            return header, err
        }
    }

    n, err = io.ReadFull(conn, header[:])
    if err != nil {
        return header, err
    }
    if n != HeaderSize {
        return header, ErrPayloadSize
    }
    return header, nil
}

func readPayload(conn net.Conn, size int, duration time.Duration) ([]byte, error) {
    var (
        n       int
        err     error
        payload = make([]byte, size)
    )
    if duration > 0 {
        err = conn.SetDeadline(time.Now().Add(duration))
        if err != nil {
            return nil, err
        }
    }
    n, err = io.ReadFull(conn, payload)
    if n != size || err != nil {
        return nil, err
    }
    return payload, nil
}

func headerGetPayloadSize(header *[HeaderSize]byte) int {
    val := binary.BigEndian.Uint32(header[4:8])
    return int(val)
}

func headerPutPayloadSize(header *[HeaderSize]byte, size int) {
    binary.BigEndian.PutUint32(header[4:8], uint32(size))
}

func headerValidMagic(header [HeaderSize]byte) bool {
    return header[0] == 'B' &&
        header[1] == 'A' &&
        header[2] == 'B' &&
        header[3] == '@'
}

func headerPutMagic(header *[HeaderSize]byte) {
    header[0] = 'B'
    header[1] = 'A'
    header[2] = 'B'
    header[3] = '@'
}

func headerTypeCode(header [HeaderSize]byte) byte {
    return header[8]
}

func headerPutTypeCode(header *[HeaderSize]byte, code byte) {
    header[8] = code
}

func headerCrc(header [HeaderSize]byte) uint32 {
    return binary.BigEndian.Uint32(header[10:])
}

func headerPutCrc(header *[HeaderSize]byte, data []byte) error {
    if data == nil || len(data) == 0 {
        binary.BigEndian.PutUint32(header[10:], 0)
        return nil
    }
    crcNum, err := CalcCrc(data)
    if err != nil {
        return err
    }
    binary.BigEndian.PutUint32(header[10:], crcNum)
    return nil
}

func CalcCrc(data []byte) (uint32, error) {
    c := crc32.NewIEEE()
    _, err := c.Write(data)
    if err != nil {
        return 0, err
    }
    return c.Sum32(), nil
}

func buildRequest(reqId uint32, service string, payload interface{}) (*pb.Message, error) {
    req := &pb.Message{}
    req.Action = proto.Int32(1)
    req.Name = proto.String(service)
    req.Id = proto.Uint32(reqId)
    if data, err := Marshal(payload); err != nil {
        return nil, err
    } else {
        req.Payload = data
    }

    return req, nil
}

func makePkt(typeCode byte, msg *pb.Message) ([]byte, error) {
    var (
        payload []byte
        err     error
    )
    if msg != nil {
        payload, err = proto.Marshal(msg)
        if err != nil {
            return nil, err
        }
    }

    header := [HeaderSize]byte{}
    headerPutMagic(&header)
    headerPutTypeCode(&header, typeCode)
    headerPutPayloadSize(&header, len(payload))
    err = headerPutCrc(&header, payload)
    if err != nil {
        return nil, err
    }
    buf := bytes.NewBuffer(header[:])
    buf.Write(payload)
    return buf.Bytes(), nil
}

func checkFunc(fn interface{}) (in []reflect.Value, f reflect.Value, ok bool) {
    defer func() {
        if e := recover(); e != nil {
            buffer := bytes.NewBufferString(fmt.Sprint(e))
            //打印调用栈信息
            buf := make([]byte, 2048)
            n := runtime.Stack(buf, false)
            stackInfo := fmt.Sprintf("\n%s", buf[:n])
            buffer.WriteString(fmt.Sprintf("panic stack info %s", stackInfo))
            log.Println(buffer)
        }
    }()
    // 检查传入的函数是否符合格式要求
    var (
        typeOfError = reflect.TypeOf((*error)(nil)).Elem()
    )
    f, ok = fn.(reflect.Value)
    if !ok {
        f = reflect.ValueOf(fn)
    }
    t := f.Type()
    if t.NumIn() != 3 { // context/request/response
        return
    }
    if t.NumOut() != 1 {
        return
    }
    if returnType := t.Out(0); returnType != typeOfError {
        return
    }
    r := t.In(1)
    w := t.In(2)

    ctx := context.Background()
    t0 := reflect.ValueOf(ctx)
    t1 := reflect.New(r.Elem())
    t2 := reflect.New(w.Elem())

    in = []reflect.Value{
        t0, t1, t2,
    }
    ok = true
    return
}
