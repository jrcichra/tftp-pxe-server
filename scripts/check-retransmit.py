#!/usr/bin/env python3
"""Regression check for the single-port retransmit-timer bug.

A stalled Pi re-ACKs the last block every ~2s. Upstream pin/tftp restarted its
retransmit timer on every packet, so DATA was never resent. This client ACKs
block 0, then never ACKs block 1 but sends duplicate ACK 0 every 0.4s, and
expects DATA 1 to be retransmitted. Also expects the 1468-byte block size.

  tftp-pxe-server -directory D -single-port -port 6969 -timeout 1 -blocksize 1468 &
  D/default/blob must be >= 4096 bytes.   python3 -I scripts/check-retransmit.py 6969
"""
import socket, struct, sys, time
port = int(sys.argv[1])
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.settimeout(0.1)
srv = ("127.0.0.1", port)
s.sendto(b"\0\1blob\0octet\0blksize\x001468\0tsize\0" + b"0\0", srv)
def recv():
    try: return s.recvfrom(4096)[0]
    except socket.timeout: return None
oack = None
for _ in range(30):
    oack = recv()
    if oack: break
assert oack and oack[:2] == b"\0\6", f"no OACK: {oack}"
assert b"1468" in oack, f"blksize not 1468: {oack}"
s.sendto(struct.pack("!HH", 4, 0), srv)
data1 = []
t0 = time.time(); last_dup = 0
while time.time() - t0 < 6:
    p = recv()
    if p and p[:2] == b"\0\3" and struct.unpack("!H", p[2:4])[0] == 1: data1.append(time.time() - t0)
    if time.time() - last_dup > 0.4:
        s.sendto(struct.pack("!HH", 4, 0), srv); last_dup = time.time()   # dup ACK, never ACK 1
print("DATA 1 seen at", [round(x, 2) for x in data1])
assert len(data1) >= 3, "server did not retransmit while duplicate ACKs kept arriving"
print("OK")
