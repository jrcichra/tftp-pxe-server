# tftp-pxe-server

A modern TFTP PXE server designed for booting multiple Raspberry Pis.
This works regardless if you're using a Pi 3B (non-plus) with the OTP bit set or a modern Raspberry Pi 4.

This wouldn't be possible without the awesome https://github.com/pin/tftp library!

By default, `tftp-pxe-server` serves files within a subdirectory based on the IP address of the requestor.

Instead of booting the Pi and figuring out the serial number for subdirectories, make boot subdirectories based on the static IP address you have in your DHCP server. Raspberry Pi 3B's don't ask for a TFTP path under their serial number, and not everyone uses `dnsmasq-tftp` in proxy mode for mac address mode.

# Getting Started

NOTE: `10.0.0.5` asking for `bootcode.bin` by default will read `./10.0.0.5/bootcode.bin`.

Use the pre-built docker image like so:

```
docker run --name=tftp-pxe-server \
    -it -d \
    --restart=unless-stopped \
    --network=host \
    -v $PWD/tftproot:/tftproot
    ghcr.io/jrcichra/tftp-pxe-server
    -directory /tftproot
```

Change the left side of `-v` to your TFTP root directory.

# Help

```
Usage of ./tftp-pxe-server:
  -directory string
        directory to serve (default ".")
  -ipPaths
        prepend request paths with src IP address (default true)
  -port int
        tftp port (default 69)
  -timeout int
        seconds for tftp timeouts (default 10)
```
