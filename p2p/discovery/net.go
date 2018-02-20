/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
)

func getUDPConn(addr *net.UDPAddr) *net.UDPConn {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Info(err.Error())
	}

	return conn
}

func getAddr(port string) *net.UDPAddr {
	address := ":" + port
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Error(err.Error())
	}

	return addr
}
