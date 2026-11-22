# Pi-like TFTP client: ACK every in-order block, ACK duplicates with their own number,
# ignore future blocks, re-ACK the last good block after 2s of silence, give up after 30s.
import socket, struct, sys, time, hashlib
port, name, outpath = int(sys.argv[1]), sys.argv[2], sys.argv[3]
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.settimeout(0.05)
srv = ("127.0.0.1", port)
rrq = b"\0\1" + name.encode() + b"\0octet\0blksize\x001468\0tsize\x000\0"
s.sendto(rrq, srv)
t0 = time.time(); last_rx = time.time(); expect = 1; peer = None; blk = 1468; out = bytearray(); done = False
def ack(n): s.sendto(struct.pack("!HH", 4, n & 0xffff), peer)
while time.time() - t0 < 30 and not done:
    try: p, a = s.recvfrom(65535)
    except socket.timeout:
        if peer and time.time() - last_rx > 2: ack(expect - 1); last_rx = time.time()
        elif not peer and time.time() - last_rx > 2: s.sendto(rrq, srv); last_rx = time.time()
        continue
    op = struct.unpack("!H", p[:2])[0]; peer = a
    if op == 6:   # OACK
        for kv in p[2:].split(b"\0"):
            pass
        parts = p[2:].split(b"\0")
        for i in range(0, len(parts) - 1, 2):
            if parts[i].lower() == b"blksize": blk = int(parts[i + 1])
        ack(0); last_rx = time.time()
    elif op == 3:
        n = struct.unpack("!H", p[2:4])[0]; d = p[4:]
        if n == expect & 0xffff:
            out += d; ack(n); expect += 1; last_rx = time.time()
            if len(d) < blk: done = True
        elif n < (expect & 0xffff): ack(n)          # duplicate: ACK it by its own number
    elif op == 5: print("ERR", p[4:60]); break
el = time.time() - t0
open(outpath, "wb").write(out)
print(f"{'DONE' if done else 'INCOMPLETE'} {len(out)} bytes in {el:.2f}s = {len(out)/1024/max(el,1e-3):.0f} KB/s")
