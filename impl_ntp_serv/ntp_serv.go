/*
	golang practice ntp server implementation
	2017.7.18 by ahui
	learn goals: udp, ntp protocol

	references:
	https://github.com/Tipoca/ntplib/blob/master/ntplib.py
*/
package main

import (
	"os"
	"fmt"
	"net"
	"time"
	"bytes"
	"encoding/binary"
)

type NTPPacket struct {
	leap byte
	version byte
	mode byte
	stratum uint8
	poll uint8
	precision int8
	rootDelay uint32
	rootDispersion uint32
	refId uint32
	refTimestamp uint64
	origTimestamp uint64
	origTimestampHigh uint16
	origTimestampLow uint16
	recvTimestamp uint64
	txTimestamp uint64
	txTimestampHigh uint16
	txTimestampLow uint16	
}

func PackPacket(packet NTPPacket) []byte {

	buf := new(bytes.Buffer)

	b1 := (packet.leap << 6 | packet.version << 3 | packet.mode)
	binary.Write(buf, binary.BigEndian, uint8(b1))

	binary.Write(buf, binary.BigEndian, uint8(packet.stratum))
	binary.Write(buf, binary.BigEndian, uint8(packet.poll))
	binary.Write(buf, binary.BigEndian, int8(packet.precision))

	binary.Write(buf, binary.BigEndian, uint32(packet.rootDelay))
	binary.Write(buf, binary.BigEndian, uint32(packet.rootDispersion))
	binary.Write(buf, binary.BigEndian, uint32(packet.refId))
	binary.Write(buf, binary.BigEndian, uint64(packet.refTimestamp))
	binary.Write(buf, binary.BigEndian, uint64(packet.origTimestamp))
	binary.Write(buf, binary.BigEndian, uint64(packet.recvTimestamp))
	binary.Write(buf, binary.BigEndian, uint64(packet.txTimestamp))


	data := buf.Bytes()
	//fmt.Printf("PackPacket=[%v] len=%v\n", buf,  len(data))

	return data
}

func UnpackPacket(data []byte) NTPPacket {
	packet := NTPPacket{}

	buf := bytes.NewReader(data)

	var b1 uint8
	var stratum uint8
	var poll uint8
	var precision int8
	var rootDelay uint32
	var rootDispersion uint32
	var refId uint32
	var refTimestamp uint64
	var origTimestamp uint64
	var recvTimestamp uint64
	var txTimestamp uint64

	binary.Read(buf, binary.BigEndian, &b1)
	binary.Read(buf, binary.BigEndian, &stratum)
	binary.Read(buf, binary.BigEndian, &poll)
	binary.Read(buf, binary.BigEndian, &precision)
	binary.Read(buf, binary.BigEndian, &rootDelay)
	binary.Read(buf, binary.BigEndian, &rootDispersion)
	binary.Read(buf, binary.BigEndian, &refId)
	binary.Read(buf, binary.BigEndian, &refTimestamp)
	binary.Read(buf, binary.BigEndian, &origTimestamp)
	binary.Read(buf, binary.BigEndian, &recvTimestamp)
	binary.Read(buf, binary.BigEndian, &txTimestamp)
	
	packet.stratum = stratum
	packet.poll = stratum
	packet.precision = precision
	packet.refId = refId
	packet.refTimestamp = refTimestamp
	packet.origTimestamp = origTimestamp
	packet.recvTimestamp = recvTimestamp
	packet.txTimestamp = txTimestamp

	return packet
}


func CheckErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func main() {

	port := 16384

	servAddr, _err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	CheckErr(_err)

	fmt.Println("starting ntp server...", servAddr)

	serv, _err := net.ListenUDP("udp", servAddr)
	CheckErr(_err)

	defer serv.Close()

	buf := make([]byte, 256)

	for {
		fmt.Println("loop...")
		nRecv, addr, _err := serv.ReadFromUDP(buf)
		//fmt.Println("_err...", _err)
		
		fmt.Println("ReadFromUDP", nRecv, addr)
		CheckErr(_err)
		recvData := buf[:nRecv]
		//fmt.Printf("recv=[%s]\n",  string(recvData))

		recvPacket := UnpackPacket(recvData)

		fmt.Printf("recvPacket=%v\n", recvPacket)

		ts := uint64(time.Now().UnixNano() / int64(time.Second))
		fmt.Println("ts", ts)

		sendPacket := NTPPacket{}
		sendPacket.stratum = 2
		sendPacket.poll = 10
		sendPacket.refTimestamp = ts
		sendPacket.recvTimestamp = ts - 5
		sendPacket.txTimestamp = ts

		sendData := PackPacket(sendPacket)
		fmt.Println("sendData", addr, len(sendData), sendData)
		serv.WriteToUDP(sendData, addr)


	}

}
