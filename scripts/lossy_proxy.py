import socket, select, random, sys, time
listen_port, server_port, loss = int(sys.argv[1]), int(sys.argv[2]), float(sys.argv[3])
random.seed(7)
front = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); front.bind(("127.0.0.1", listen_port))
clients = {}   # client addr -> upstream socket
socks = {}     # upstream socket -> client addr
srv = ("127.0.0.1", server_port)
stats = {"fwd": 0, "drop": 0}
while True:
    r, _, _ = select.select([front] + list(socks), [], [], 1)
    for s in r:
        data, addr = s.recvfrom(65535)
        stats["fwd"] += 1
        if random.random() < loss:
            stats["drop"] += 1; continue
        if s is front:
            up = clients.get(addr)
            if up is None:
                up = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); clients[addr] = up; socks[up] = addr
            up.sendto(data, srv)
        else:
            front.sendto(data, socks[s])
