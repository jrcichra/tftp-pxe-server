#!/bin/bash
# Downloads a 1.5MB file through a lossy UDP proxy with a Pi-like client, lockstep vs -window.
# usage: scripts/check-lossy.sh  (needs go, python3)   Expect every run "DONE ... [identical]".
set -u
cd "$(dirname "$0")/.."; T=$(mktemp -d); mkdir -p "$T/root/default"
go build -o "$T/srv" ./cmd/server || exit 1
head -c 1500000 /dev/urandom > "$T/root/default/f"
n=0
for cfg in "0 0.11" "4 0.11" "8 0.2"; do set -- $cfg; n=$((n+1)); sp=$((7500+n*10))
  "$T/srv" -directory "$T/root" -single-port -port $sp -metrics-port $((sp+2000)) -retransmit 5ms -blocksize 1468 -retries 6000 -window $1 >/dev/null 2>&1 & S=$!
  python3 -I scripts/lossy_proxy.py $((sp+1)) $sp $2 & P=$!; sleep 1
  r=$(timeout 60 python3 -I scripts/pi_client.py $((sp+1)) f "$T/out"); cmp -s "$T/root/default/f" "$T/out" && ok=identical || ok=MISMATCH
  echo "window=$1 loss=$2: $r [$ok]"; kill -9 $S $P 2>/dev/null; wait $S $P 2>/dev/null
done; rm -rf "$T"
