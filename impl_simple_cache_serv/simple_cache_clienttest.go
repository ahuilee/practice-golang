package main

import (

	"fmt"
	"net"
	"github.com/satori/go.uuid"
	"./simple_cache"
)


type SimpleCacheClient struct {
	conn *net.Conn
	recvPackets chan []interface{}
}

func (client *SimpleCacheClient) Connect(host string, port int) {

	fmt.Printf("connect...%s:%v\n", host, port)
	conn, _err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if _err != nil {
		fmt.Println(_err)
		return
	}
	client.conn = &conn
	go client.RunRead()
}

func (client *SimpleCacheClient) RunRead() {
	
	fmt.Println("client >> RunRead..........")
	
	recvBuf := make([]byte, 4096)
	receiver := simple_cache.DataReceiver{}	


	for {

		nRecv, _err :=  (*client.conn).Read(recvBuf)
		//fmt.Println("nRecv=>", nRecv)
		if _err != nil {
			fmt.Println("err", _err)
			break
		}

		if nRecv < 1 {
			break
		}

		packets := receiver.DataReceived(recvBuf[:nRecv])

		for _, packet := range packets {
			//fmt.Println("ForEach packet", packet)
			client.recvPackets <- packet
		}	
	}
}

func (client *SimpleCacheClient) SetValue(key string, value string) {
	//fmt.Printf("SetValue key=%s val=%s\n", key, value)

	(*client.conn).Write(simple_cache.PackArrayPacket("SET", key, value))
}

func (client *SimpleCacheClient) GetValue(key string) string {
	//fmt.Printf("GetValue key=%s\n", key)

	(*client.conn).Write(simple_cache.PackArrayPacket("GET", key))

	packet := <-client.recvPackets

	//fmt.Println("GetValue", key, "recvPackets", packet)

	return packet[0].(string)			
}

func NewClient(host string, port int) *SimpleCacheClient {
	client := SimpleCacheClient{}
	recvPackets := make(chan []interface{})
	
	client.recvPackets = recvPackets

	client.Connect(host, port)

	return &client
}

func main() {

	client := NewClient("127.0.0.1", 16384)	

	var val string

	for i:=0; i<65535; i++ {
		val = fmt.Sprintf("%v", uuid.NewV4())

		key := fmt.Sprintf("key-%v", uuid.NewV4())
		
		client.SetValue(key, val)
		val2 := client.GetValue(key)
		fmt.Printf("%06d TEST %v %v\n", i, key, val2)
		if val != val2 {
			err := fmt.Errorf("%v != %v", val, val2)
			fmt.Println(err)
			break
			
		}
	
	}

}
