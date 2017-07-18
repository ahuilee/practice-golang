import os, sys
import socket
import time
import ntplib

def main():
    #sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    #sock.sendto("hello!", ("127.0.0.1", 1234))
    c = ntplib.NTPClient()
    rep = c.request("localhost", port=16384)
    rep = c.request("pool.ntp.org")
    print "version", rep.version
    print "offset", rep.offset
    print "root_delay", rep.root_delay
    print "ref_id", rep.ref_id
    print "ref_id text", ntplib.ref_id_to_text(rep.ref_id)
    print "tx_time", rep.tx_time, time.time()
    print "ctime", time.ctime(rep.tx_time)


if __name__ == "__main__":
    main()

