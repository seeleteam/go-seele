/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"encoding/binary"
	"log"
)

// ConvertInt64ToBytes converts a int64 value into a byte array
func ConvertInt64ToBytes(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
