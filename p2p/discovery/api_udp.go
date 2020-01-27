package discovery

import "net"

type PrivateUdpAPI struct {
	u *udp
}

func NewPrivateUdpAPI(u *udp) *PrivateUdpAPI {
	return &PrivateUdpAPI{u}
}

func (api *PrivateUdpAPI) GetUdpServer() *udp {
	return api.u
}

// GetConnPeer get upd conn node
func (api *PrivateUdpAPI) GetConnPeer() *net.UDPConn {
	return api.u.conn
}

// GetSelfID get self node
func (api *PrivateUdpAPI) GetSelfID() *Node {
	return api.u.self
}

// func (api *PrivateUdpAPI) GetReply() *reply {
// 	if api.u.gotReply != nil {
// 		return
// 	}
// }
