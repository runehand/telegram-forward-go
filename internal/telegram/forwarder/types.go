package forwarder

import "github.com/gotd/td/tg"

type target struct {
	Name string
	Peer tg.InputPeerClass
}