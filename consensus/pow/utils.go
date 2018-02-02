package consensus

import (
    "bytes"
    "encoding/binary"
    "log"
)

// Converts a int64 value into a byte array
func Int2Hex(num int64) []byte {
    buff := new(bytes.Buffer)
    err := binary.Write(buff, binary.BigEndian, num)
    if err != nil {
        log.Panic(err)
    }

    return buff.Bytes()
}
