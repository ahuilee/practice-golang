package simple_cache

import (
	"fmt"
	"strconv"
	"strings"
	"reflect"
)

const PACKET_STATE_NONE int = 0
const PACKET_STATE_MULTI int = 1

const PACKET_ELE_STATE_TYPE int = 0
const PACKET_ELE_STATE_VALUE int = 1

const PACKET_ELETYPE_NONE int = 0
const PACKET_ELETYPE_STR int = 1


type DataReceiver struct {
	buf string
	pktState int
	pktEleState int
	pktEleType int
	pktMultiResults []interface{}
	pktMultiResultCount int
	pktEleStrSize int
}

func New(receiver DataReceiver) DataReceiver {
	fmt.Println("DataReceiver::New")
	
	self := DataReceiver{}

	self.pktState = PACKET_STATE_NONE
	self.pktEleState = PACKET_ELE_STATE_TYPE
	
	self.pktEleType = PACKET_ELETYPE_NONE
	self.pktEleStrSize = 0
	self.pktMultiResults = make([]interface{}, 0)
	self.pktMultiResultCount = 0
	return self
}

func (self *DataReceiver) DataReceived(data []byte) [][]interface{} {
	//receiver.buf += string(recvBuf[:nRecv])
	self.buf += string(data)

	var packets [][]interface{}

	//fmt.Println("DataReceived=", string(data))
	
	for {
		
		newLineIdx := strings.Index(self.buf, "\r\n")
		if newLineIdx < 0 {
			break
		}			

		switch self.pktState {
		case PACKET_STATE_NONE:
			line := self.buf[:newLineIdx]
			self.buf = self.buf[newLineIdx+2:]
			//fmt.Println("line", line)

			if line[:1] == "*" {
				self.pktState = PACKET_STATE_MULTI
				self.pktEleState = PACKET_ELE_STATE_TYPE
				pktMultiCount, _ := strconv.Atoi(line[1:])
				self.pktMultiResultCount = 0
				self.pktMultiResults = make([]interface{}, pktMultiCount)
			}
			break
			
		case PACKET_STATE_MULTI: 

			switch self.pktEleState {
			case PACKET_ELE_STATE_TYPE:
				line := self.buf[:newLineIdx]
				self.buf = self.buf[newLineIdx+2:]

				if line[:1] == "s" {
					self.pktEleType = PACKET_ELETYPE_STR
					self.pktEleState = PACKET_ELE_STATE_VALUE
					self.pktEleStrSize, _ = strconv.Atoi(line[1:])
					self.pktEleStrSize += 2

				}

				break
			case PACKET_ELE_STATE_VALUE:
				switch self.pktEleType {
				case PACKET_ELETYPE_STR:
					if len(self.buf) < self.pktEleStrSize {
						break
					}

					self.pktEleState = PACKET_ELE_STATE_TYPE
					strVal := self.buf[:self.pktEleStrSize-2]
					self.buf = self.buf[self.pktEleStrSize:]

					self.pktMultiResults[self.pktMultiResultCount] = strVal

					//fmt.Println(">> pktMultiResults", self.pktMultiResultCount, "strVal", strVal)

					self.pktMultiResultCount += 1

					if self.pktMultiResultCount >= len(self.pktMultiResults) {
						self.pktState = PACKET_STATE_NONE
						//handler.PacketReceived(pktMultiResults)
						//fmt.Println("pktMultiResults", self.pktMultiResults)
						packets = append(packets, self.pktMultiResults)
					}

					break
				}

				break
			}

			break
		}
	}

	return packets
}


func PackArrayPacket(args ...interface{}) []byte {

	packet := ""
	packet += fmt.Sprintf("*%d\r\n", len(args))

	for _, arg := range args {
		arg_type := reflect.TypeOf(arg).Kind()
		//fmt.Println("type", arg_type)
		if arg_type == reflect.String {
			arg_str := arg.(string)
			packet += fmt.Sprintf("s%d\r\n", len(arg_str))
			packet += fmt.Sprintf("%s\r\n", arg_str)		
		}
	}

	return []byte(packet)
}
