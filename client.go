package b2b

import "net"

type client struct {
	conn net.Conn
}

func (c *client) handle() {

}
