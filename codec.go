package rpc

import (
    "github.com/vmihailenco/msgpack"
)

func Marshal(ptr interface{}) ([]byte, error) {
    return msgpack.Marshal(ptr)
}

func Unmarshal(data []byte, ptr interface{}) error {
    return msgpack.Unmarshal(data, ptr)
}
