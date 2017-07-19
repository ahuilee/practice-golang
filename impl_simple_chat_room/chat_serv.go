package main

import (
	"os"
	//"io"
	"fmt"
	"time"
	//"errors"
	"net"
	"net/http"
	"strconv"
	"reflect"
	"html/template"
	"bytes"
	"encoding/binary"
	"encoding/json"
	//"container/list"
	"github.com/satori/go.uuid"
	"golang.org/x/net/websocket"
)

const OPCODE_PACKET uint8 = 1
const DATATYPE_INT uint8 = 1
const DATATYPE_STR uint8 = 2
const DATATYPE_ARR uint8 = 3

const COMMAND_AUTH string = "AUTH"
const COMMAND_DISPLAY_TEXT string = "DISPLAY_TEXT"
const COMMAND_USERLIST_CHANGED string = "USERLIST_CHANGED"
const COMMAND_USERLIST string = "USERLIST"
const COMMAND_TALK string = "TALK"

type Packet struct {
	values []interface{}
	err error
}

type Connection struct {
	id int
	ws *websocket.Conn
	serv *ChatServ
	user *ChatUser
	room *ChatRoom
	sendData chan []interface{}
}

type ChatServ struct {
	lastId int
	getRoomById map[int]*ChatRoom
	connections map[int]*Connection
}

type ChatUser struct {
	key string
	nickname string
	conn *Connection
}

type ChatHttpHandler struct {
	serv *ChatServ
}

type ChatRoom struct {
	id int
	name string
	users map[string] *ChatUser
}

type TalkContext struct {
	Text string
	Color string
}

func NewChatServ() *ChatServ {

	serv := new(ChatServ)
	serv.lastId = 0
	
	serv.getRoomById = make(map[int]*ChatRoom)
	serv.connections = make(map[int]*Connection)
	return serv
}


func (s *ChatServ) NewConnection(ws *websocket.Conn) *Connection {
	id := s.MakeId()
	conn := new(Connection)
	conn.id = id
	conn.ws = ws
	conn.serv = s
	conn.sendData = make(chan []interface{})
	s.connections[id] = conn
	return conn
}

func (s *ChatServ) MakeId() int {
	id := s.lastId + 1
	s.lastId = id
	return id
}



func (u *ChatUser) Key() string {
	return u.key
}

func (r *ChatRoom) Name() string {
	return r.name
}

func (r *ChatRoom) Id() int {
	return r.id
}


func (r *ChatRoom) BroadcastPacket(args ...interface{}) {
	fmt.Println("BroadcastPacket", args)
	data := PackPacket(args...)
	for _, u := range r.users {
		u.conn.sendData <- data
	}
}


func (serv *ChatServ) CreateRoom(name string) {
	id := serv.lastId + 1
	serv.lastId = id
	room := new(ChatRoom)
	room.id = id
	room.name = name
	room.users = make(map[string] *ChatUser)

	//serv.rooms.PushBack(room)
	serv.getRoomById[id] = room
}

func HandleConn(conn *net.Conn) {

	buf := make([]byte, 1024)

	for {
		nRecv, _err := (*conn).Read(buf)
		if _err != nil {
			fmt.Println(_err)
			break
		}
		fmt.Println("nRecv", nRecv)
		fmt.Printf("recv=[%v]\n", string(buf[:nRecv]))
	}
}

func (serv *ChatServ) Run(port uint16) {

	fmt.Println("starting server...", port)

	listener, _err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	CheckErr(_err)

	for {
		conn, _err := listener.Accept()
		CheckErr(_err)
		fmt.Println("ACCEPT", conn)
		go HandleConn(&conn)
	}

}


func CheckErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}


func main() {

	chatServ := NewChatServ()
	chatServ.CreateRoom("hello!")
	chatServ.CreateRoom("hello!2")

	chatHttpHandler := new(ChatHttpHandler)
	chatHttpHandler.serv = chatServ

	staticFileServ := http.FileServer(http.Dir("./static"))

	http.Handle("/static/", http.StripPrefix("/static/", staticFileServ))
	http.HandleFunc("/", chatHttpHandler.HandleIndex)
	http.HandleFunc("/join", chatHttpHandler.HandleJoin)
	http.Handle("/chat/", websocket.Handler(chatHttpHandler.HandleChat))

	http.ListenAndServe(":8000", nil)
}

func UnpackPacket(data []byte) []interface{} {
	var packet []interface{}
	buf := bytes.NewReader(data)
	
	
		var argCount uint16 
		var argType uint8
		var strLen uint16
		var intVal int32

		binary.Read(buf, binary.BigEndian, &argCount)

		packet = make([]interface{}, argCount)
		var i uint16

		for i=0; i<argCount; i++ {
			binary.Read(buf, binary.BigEndian, &argType)
			switch(argType) {
			case DATATYPE_INT:
				binary.Read(buf, binary.BigEndian, &intVal)
				packet[i] = intVal

				break
			case DATATYPE_STR:
				binary.Read(buf, binary.BigEndian, &strLen)
				strBuf := make([]byte, strLen)
				nReaded, _err := buf.Read(strBuf)
				if _err != nil {
					fmt.Println("err", _err)
				}
				fmt.Println("nReaded", nReaded)
				strVal := string(strBuf)
				fmt.Println("strVal", strVal)
				packet[i] = strVal
				break
			}
		}

	

	return packet
}


func PackArray(args ...interface{}) []byte {
	buf := new(bytes.Buffer)

	fmt.Println("PackArray", len(args), args)
	
	binary.Write(buf, binary.BigEndian, uint16(len(args)))

	for _, arg := range args {
		val := reflect.ValueOf(arg)
		argType := reflect.TypeOf(arg)
		argTypeKind := argType.Kind()
		//fmt.Println("PackArray argType", argType, argTypeKind)
		switch argTypeKind {
			case reflect.String:
				binary.Write(buf, binary.BigEndian, DATATYPE_STR)
				strVal := val.String()
				binary.Write(buf, binary.BigEndian, uint16(len(strVal)))
				binary.Write(buf, binary.BigEndian, []byte(strVal))

			case reflect.Int:
				binary.Write(buf, binary.BigEndian, DATATYPE_INT)
				intVal := val.Int()
				binary.Write(buf, binary.BigEndian, intVal)

				break
			case reflect.Slice:

				binary.Write(buf, binary.BigEndian, DATATYPE_ARR)

				arrVal := make([]interface{}, val.Len())

				for i:=0; i<val.Len(); i++ {
					ele := val.Index(i)
					switch ele.Kind() {
					case reflect.String:
						arrVal[i] = ele.String()
						break
					}
				}

				fmt.Println("PackArray argType.Slice", arrVal)

				arrBytes := PackArray(arrVal...)
				binary.Write(buf, binary.BigEndian, uint32(len(arrBytes)))
				binary.Write(buf, binary.BigEndian, arrBytes)

				break
		}
	}

	return buf.Bytes()
}
/*
func PackPacket(args ...interface{}) []byte {

	arrData := PackArray(args...)
	arrLen := uint32(len(arrData))

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, OPCODE_PACKET)
	binary.Write(buf, binary.BigEndian, arrLen)
	binary.Write(buf, binary.BigEndian, []byte("\x00\x00\x00"))
	binary.Write(buf, binary.BigEndian, arrData)

	return buf.Bytes()
}*/
func PackPacket(args ...interface{}) []interface{} {

	return args
}


func (conn *Connection) PacketReceived(packet []interface{}) {

	//rd := NewWebSockReader(conn.ws)

	fmt.Println("ReadPacket", packet)


	cmdName := packet[0].(string)
	fmt.Println("ReadPacket", cmdName, packet)

	switch cmdName{
	case COMMAND_AUTH:
		roomId := int(packet[1].(float64))
		authKey := packet[2].(string)
		room := conn.serv.getRoomById[roomId]
		user := room.users[authKey]
		user.conn = conn
		conn.user = user
		conn.room = room
		fmt.Println("authKey", authKey, user)
		room.BroadcastPacket(COMMAND_USERLIST_CHANGED)
		room.BroadcastPacket(COMMAND_DISPLAY_TEXT, fmt.Sprintf("%v 加入聊天", user.nickname))
		break
	case COMMAND_USERLIST:
		userlist := make([]map[string]string, len(conn.room.users))
		i := 0
		for _, u := range conn.room.users {
			userlist[i] = map[string]string{"name": u.nickname, "key": u.key}
			i += 1
		}
		_userlistJsonStr, _ := json.Marshal(userlist)

		fmt.Println("userlist jsonStr", string(_userlistJsonStr))
		conn.sendData <- PackPacket(COMMAND_USERLIST, string(_userlistJsonStr))
		break
	case COMMAND_TALK:
	
		talkCtx := packet[1].(map[string]interface{})
		
		now := time.Now()

		displayText := fmt.Sprintf(`[%02d:%02d] %v說: <span style="color: %v">%v</span>`, now.Hour(), now.Minute(), conn.user.nickname, talkCtx["color"], talkCtx["text"])
		conn.room.BroadcastPacket(COMMAND_DISPLAY_TEXT, displayText)
		break
	}

}

func (conn *Connection) HandleWrite() {

	for {

		
		fmt.Println("HandleWrite...")
		select {
		case packet := <-conn.sendData:
			fmt.Println("HandleWrite", len(packet), packet)
			websocket.JSON.Send(conn.ws, packet)

		}
		//conn.ws.Write(data)
	}

}

func (handler *ChatHttpHandler) HandleChat(ws *websocket.Conn) {

	conn := handler.serv.NewConnection(ws)

	go conn.HandleWrite()

	fmt.Println(">> HandleChat", conn)

	conn.sendData <- PackPacket(COMMAND_AUTH)	
	

	for {
		var values []interface{}

		fmt.Println("websocket.JSON.Receive...")
		websocket.JSON.Receive(conn.ws, &values)
		fmt.Println("websocket.JSON.Receive", values)
		if len(values) == 0 {
			break
		}
		conn.PacketReceived(values)
	}

	if conn.room != nil {

		delete(conn.room.users, conn.user.key)
		conn.room.BroadcastPacket(COMMAND_USERLIST_CHANGED)
		conn.room.BroadcastPacket(COMMAND_DISPLAY_TEXT, fmt.Sprintf("%v 離開了聊天室", conn.user.nickname,))
	}
}


func (handler *ChatHttpHandler) HandleJoin(w http.ResponseWriter, req *http.Request) {


	nickname := req.FormValue("nickname")
	roomId, _ := strconv.Atoi(req.FormValue("key"))

	room := handler.serv.getRoomById[roomId]

	fmt.Println("HandleJoin", nickname, roomId, room)

	key := fmt.Sprintf("%v", uuid.NewV4())

	user := new(ChatUser)
	user.key = key
	user.nickname = nickname
	room.users[key] = user

	fmt.Println("User", user)

	context := make(map[string]interface{})
	context["user"] = user
	context["room"] = room
	context["hatServUrl"] = fmt.Sprintf("127.0.0.1:8000/chat/?q=%v", key)


	t, _ := template.ParseFiles("./templates/chat.html")
	t.Execute(w, context)
}

func (handler *ChatHttpHandler) HandleIndex(w http.ResponseWriter, request *http.Request) {

	//fmt.Println("HandleIndex", request)

	context := make(map[string]interface{})
	rooms := make([]*ChatRoom, len(handler.serv.getRoomById))

	i := 0

	for _, room := range handler.serv.getRoomById {
		rooms[i] = room
		i += 1
	}

	fmt.Println("handler.serv.rooms", rooms)

	context["rooms"] = rooms
	context["title"] = "test"

	t, _ := template.ParseFiles("./templates/index.html")
	t.Execute(w, context)
}
