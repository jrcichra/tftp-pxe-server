package tftp

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// readFromWindow is a sliding-window replacement (jrcichra/tftp-pxe-server) for the
// upstream "anticipate" sender, used when Server.SetAnticipate(winsz>1) is on.
//
// Upstream sends winsz blocks and then needs the ACKs for exactly n, n+1, ... n+winsz-1 to
// arrive in order within one pass. With any packet loss the ACKs of a resent window arrive
// scattered and it livelocks (seen on a Pi 5 with ~11% loss: block 25-28 resent forever).
//
// Here an ACK is cumulative: ACK n acknowledges every outstanding block up to n and slides the
// window; the window is refilled immediately. On a timeout the whole outstanding window is
// resent (go-back-N). The retransmit timer only restarts on progress or on a resend, so
// duplicate ACKs from a stalled client cannot starve it.
func readFromWindow(s *sender, r io.Reader) (n int64, err error) {
	type pkt struct {
		blk uint16
		buf []byte
		n   int
	}
	var window []*pkt
	winsz := int(s.sendA.winsz)
	if winsz < 2 {
		winsz = 2
	}
	blkLen := len(s.send) - 4
	nextBlk := uint16(1)
	eof := false

	fill := func() error {
		for !eof && len(window) < winsz {
			buf := make([]byte, blkLen+4)
			l, err := io.ReadFull(r, buf[4:])
			if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
				return err
			}
			binary.BigEndian.PutUint16(buf[0:2], opDATA)
			binary.BigEndian.PutUint16(buf[2:4], nextBlk)
			if l < blkLen { // short (possibly empty) block ends the transfer
				eof = true
			}
			window = append(window, &pkt{blk: nextBlk, buf: buf, n: 4 + l})
			n += int64(l)
			nextBlk++
		}
		return nil
	}

	sent := 0 // window[:sent] have been transmitted since the last (re)send of the window
	timeouts := 0
	lastActivity := time.Now()
	for {
		if err = fill(); err != nil {
			s.abort(err)
			return n, err
		}
		if len(window) == 0 {
			if s.hook != nil {
				s.hook.OnSuccess(s.buildTransferStats())
			}
			return n, nil
		}
		for sent < len(window) {
			p := window[sent]
			if err = s.conn.sendTo(p.buf[:p.n], s.addr); err != nil {
				s.abort(err)
				return n, err
			}
			s.datagramsSent++
			sent++
		}
		remaining := s.timeout - time.Since(lastActivity)
		if remaining < time.Microsecond {
			remaining = time.Microsecond
		}
		if err = s.conn.setDeadline(remaining); err != nil {
			s.abort(err)
			return n, err
		}
		nn, addr, rerr := s.conn.readFrom(s.receive)
		if rerr != nil {
			if _, ok := rerr.(net.Error); ok {
				timeouts++
				if timeouts > s.retries {
					s.abort(rerr)
					return n, rerr
				}
				s.retry.backoff()
				sent = 0 // go-back-N: resend everything outstanding
				lastActivity = time.Now()
				continue
			}
			s.abort(rerr)
			return n, rerr
		}
		if !addr.IP.Equal(s.addr.IP) || (s.tid != 0 && addr.Port != s.tid) {
			continue
		}
		p, perr := parsePacket(s.receive[:nn])
		if perr != nil {
			continue
		}
		s.tid = addr.Port
		switch p := p.(type) {
		case pACK:
			if d := p.block() - window[0].blk; int(d) < len(window) { // d is uint16: handles wrap
				window = window[int(d)+1:]
				sent -= int(d) + 1
				if sent < 0 {
					sent = 0
				}
				s.datagramsAcked++
				timeouts = 0
				lastActivity = time.Now()
			}
		case pERROR:
			err = fmt.Errorf("sending block %d: code=%d, error: %s", window[0].blk, p.code(), p.message())
			s.abort(err)
			return n, err
		}
	}
}
