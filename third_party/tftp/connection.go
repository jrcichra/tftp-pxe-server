package tftp

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv6"

	"golang.org/x/net/ipv4"
)

type connectionError struct {
	error
	timeout   bool
	temporary bool
}

func (t *connectionError) Timeout() bool {
	return t.timeout
}

func (t *connectionError) Temporary() bool {
	return t.temporary
}

type connection interface {
	sendTo([]byte, *net.UDPAddr) error
	readFrom([]byte) (int, *net.UDPAddr, error)
	setDeadline(time.Duration) error
	close()
}

type connConnection struct {
	conn *net.UDPConn
}

type chanConnection struct {
	server        *Server
	channel       chan []byte
	srcAddr, addr *net.UDPAddr
	timeout       time.Duration
	deadline      time.Time // absolute, set by setDeadline (see readFrom)
	complete      chan string
}

func (c *chanConnection) sendTo(data []byte, addr *net.UDPAddr) error {
	var err error
	c.server.Lock()
	defer c.server.Unlock()
	if conn, ok := c.server.conn.(*net.UDPConn); ok {
		srcAddr := c.srcAddr.IP.To4()
		var cmm []byte
		if srcAddr != nil {
			cm := &ipv4.ControlMessage{Src: srcAddr}
			cmm = cm.Marshal()
		} else {
			cm := &ipv6.ControlMessage{Src: c.srcAddr.IP}
			cmm = cm.Marshal()
		}
		_, _, err = conn.WriteMsgUDP(data, cmm, c.addr)
	} else {
		_, err = c.server.conn.WriteTo(data, addr)
	}
	return err
}

// PATCHED (jrcichra/tftp-pxe-server): upstream waited a fresh c.timeout on every
// call, so each packet the caller ignores (a stalled client re-ACKing the last
// block every ~2s) restarted the retransmit timer and the lost DATA packet was
// never resent. Wait until the absolute deadline set by setDeadline instead,
// matching connConnection's SetReadDeadline behaviour.
func (c *chanConnection) readFrom(buffer []byte) (int, *net.UDPAddr, error) {
	if c.deadline.IsZero() {
		c.deadline = time.Now().Add(c.timeout)
	}
	select {
	case data := <-c.channel:
		for i := range data {
			buffer[i] = data[i]
		}
		return len(data), c.addr, nil
	case <-time.After(time.Until(c.deadline)):
		return 0, nil, makeError(c.addr.String())
	}
}

func (c *chanConnection) setDeadline(deadline time.Duration) error {
	c.timeout = deadline
	c.deadline = time.Now().Add(deadline)
	return nil
}

func (c *chanConnection) close() {
	c.server.Lock()
	defer c.server.Unlock()
	close(c.channel)
	delete(c.server.handlers, c.addr.String())
}

func (c *connConnection) sendTo(data []byte, addr *net.UDPAddr) error {
	_, err := c.conn.WriteToUDP(data, addr)
	return err
}

func makeError(addr string) net.Error {
	error := connectionError{
		timeout:   true,
		temporary: true,
	}
	error.error = fmt.Errorf("Channel timeout: %v", addr)
	return &error
}

func (c *connConnection) readFrom(buffer []byte) (int, *net.UDPAddr, error) {
	return c.conn.ReadFromUDP(buffer)
}

func (c *connConnection) setDeadline(deadline time.Duration) error {
	return c.conn.SetReadDeadline(time.Now().Add(deadline))
}

func (c *connConnection) close() {
	c.conn.Close()
}
