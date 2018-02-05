/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"fmt"
	"net"
	"strconv"
	_ "time"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/common"
)

// PingInfo ping info
type PingInfo struct {
	NodeID string
}

func ListenTest(port string)  {
	conn := getUDPConn(port)
	Listen(conn)
}

func SendTest(sourcePort string, targetPort string)  {
	src := getAddr(sourcePort)
	target := getAddr(targetPort)

	Send(src, target)
}


func Listen(conn *net.UDPConn) {
	defer conn.Close()
	//for {
		data := make([]byte, 100)
		n, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Info(err)
		}

		log.Info("ip:", remoteAddr.IP, "port:", remoteAddr.Port, "network:", remoteAddr.Network,
			"zone:", remoteAddr.Zone)
		log.Info("n:", n)

		buff := data[:n]

		info := PingInfo{}
		err = common.Decoding(buff, &info)

		if err != nil {
			log.Info(err)
		}

		log.Info("nodeid:", info.NodeID)
	//}
}

func getUDPConn(port string) *net.UDPConn {
	addr := getAddr(port)

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Info(err)
	}

	return conn
}

func getAddr(port string) *net.UDPAddr {
	address := ":" + port
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Error(err)
	}

	return addr
}

func Send(sourceAddr *net.UDPAddr, targeAddr *net.UDPAddr) {
	fmt.Printf("%v, %v", sourceAddr, targeAddr)

	conn, err := net.DialUDP("udp", sourceAddr, targeAddr)
	if err != nil {
		log.Info(err)
	}
	defer conn.Close()

	i := 12323;
	//for {
		info := PingInfo{NodeID: strconv.Itoa(i)}

		buff, err:= common.Encoding(info)
		if err != nil {
			log.Info(err)
		}

		log.Info("send:", info.NodeID)

		log.Info("buff length:", len(buff))

		conn.Write(buff)
		i++

		//time.Sleep(5 * time.Second)
	//}
}
