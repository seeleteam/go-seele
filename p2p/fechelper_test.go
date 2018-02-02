/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import (
    "bytes"
    "fmt"
    "math/rand"
    "testing"
    "time"
)

func GetRandomString(length int) string {
    str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
    strbytes := []byte(str)
    strLen := len(strbytes)

    result := []byte{}
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    for i := 0; i < length; i++ {
        result = append(result, strbytes[r.Intn(strLen)])
    }
    return string(result)
}

//Test_fechelper_mem create random string in memory; simulate packet loss, and try recover by fec
func Test_fechelper_mem(t *testing.T) {
    bundleNum, fecExtLen := 16, 4
    helper := new(FECHelper)
    helper.Init(16)
    str1 := GetRandomString(16 * 1024)
    strBuf := []byte(str1)
    rcvList := make([]*VPacketItem, 0, 16)
    for i := 0; i < 16; i++ {
        p := &VPacket{
            seq:      uint32(i),
            packType: 0,
            fecIdx:   0,
        }
        p.data = make([]byte, 1024)
        copy(p.data, strBuf[i*1024:i*1024+1024])
        p.dataLen = 1024
        pItem := &VPacketItem{
            p: p,
        }
        rcvList = append(rcvList, pItem)
    }
    pFECInfo := NewFECInfo()

    for idx := 0; idx < 8; idx++ {
        pBitVec := helper.canVec[idx]
        p := new(VPacket)
        p.packType, p.fecIdx = 1, idx
        p.data = make([]byte, 1024)

        for i := 0; i < bundleNum; i++ {
            if !pBitVec.GetBit(uint(i)) {
                continue
            }
            p1 := rcvList[i].p
            for j := 0; j < 1024; j++ {
                p.data[j] = p.data[j] ^ p1.data[j]
            }
        }

        pFECInfo.fecPackets[idx] = p
    }

    //init ok, simulate packet loss
    for num := 0; num < 8; num++ {
        //round one
        r := rand.New(rand.NewSource(time.Now().UnixNano()))
        pBitTmp := new(VBitVec)
        pBitTmp.Init(uint(bundleNum))
        for i := 0; i < bundleNum; i++ {
            pBitTmp.SetBit(uint(i), r.Intn(100) >= 10)
        }
        for i := 0; i < fecExtLen; i++ {
            if r.Intn(100) >= 10 {
                pBitTmp.ExtFlag |= 1 << uint((7 - i))
            }
        }

        //recover according to pBitTmp
        for {
            pC := helper.GetRecoverInfo(pBitTmp)
            if pC == nil {
                fmt.Printf("try Recovered not %02x %02x | %02x\r\n", pBitTmp.pBuf[0], pBitTmp.pBuf[1], pBitTmp.ExtFlag)
                break
            }
            fmt.Printf("try Recovered %02x %02x | %02x", pBitTmp.pBuf[0], pBitTmp.pBuf[1], pBitTmp.ExtFlag)
            pFec := &VPacket{
                data: make([]byte, 1024),
            }
            recoverSeq := 0
            for i := 0; i < bundleNum; i++ {
                if !pC.GetBit(uint(i)) {
                    continue
                }
                if !pBitTmp.GetBit(uint(i)) {
                    recoverSeq = i
                    continue
                }
                p1 := rcvList[i].p
                for j := 0; j < 1024; j++ {
                    pFec.data[j] = pFec.data[j] ^ p1.data[j]
                }
            }

            for i := 0; i < 8; i++ {
                if !pC.GetFlagBit(uint(i)) {
                    continue
                }
                p1 := pFECInfo.fecPackets[i]
                for j := 0; j < 1024; j++ {
                    pFec.data[j] = pFec.data[j] ^ p1.data[j]
                }
            }

            pFec.seq = uint32(recoverSeq)
            pBitTmp.SetBit(uint(recoverSeq), true)

            //mem cmp
            cmpRet := bytes.Equal(pFec.data, rcvList[recoverSeq].p.data)
            fmt.Printf(" ==> recoverd seq=%d compare=%t\r\n", recoverSeq, cmpRet)
        }
    }
}
