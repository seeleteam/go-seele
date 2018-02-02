package common

import (
    "bytes"
    "encoding/binary"
    "log"
)

// Int2Bytes Converts a int64 value into a byte array
func Int2Bytes(num int64) []byte {
    buff := new(bytes.Buffer)
    err := binary.Write(buff, binary.BigEndian, num)
    if err != nil {
        log.Panic(err)
    }

    return buff.Bytes()
}