/*
	golang simple redis server implementation
	2017.7.17 by ahui
	learn goals: socket, interface, channels, sqlite

	references:
	https://redis.io/topics/protocol
*/

//package simple_redis_impl
package main

import (
	"os"
	"fmt"
	"strconv"
	"net"
	"path/filepath"
	"database/sql"
	"reflect"	
	_ "github.com/mattn/go-sqlite3"
	"./simple_cache"
)

const STORAGE_GET byte = 1
const STORAGE_SET byte = 2

type StorageQueue struct {
	que chan *StorageItem
}

type StorageItem struct {
	itemType byte
	key string
	value string
	evt chan byte
}

type ConnectionHandler struct {
	conn *net.Conn
	que *StorageQueue
}

func NewStorageQueue() *StorageQueue {

	fmt.Println("StorageQueue::New")

	que := StorageQueue{}
	que.que = make(chan *StorageItem)

	return &que
}

func (q *StorageQueue) Set(key string, value string) {
	evt := make(chan byte)
	item := StorageItem{itemType: STORAGE_SET, key: key, value: value, evt: evt}
	q.que <- &item
	
	<-evt
}

func (q *StorageQueue) Get(key string) string {
	evt := make(chan byte)
	item := StorageItem{itemType: STORAGE_GET, key: key, evt: evt}
	q.que <- &item
	
	<-evt

	return item.value
}

func (q *StorageQueue) Run() {

	dbpath := "./data/data.db"
	fullpath, _ := filepath.Abs(dbpath)
	fmt.Println("Open", fullpath)
	dirpath := filepath.Dir(fullpath)
	fmt.Println("dirpath", dirpath)

	_, err := os.Stat(dirpath)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(dirpath, 0700)
		}
	}


	
	db, _err := sql.Open("sqlite3", dbpath)
	//checkErr(err)
	if _err != nil {
		fmt.Println("err", _err)
	}
	fmt.Println("dbtype", reflect.TypeOf(db))

	stmt, _err := db.Prepare(`CREATE TABLE IF NOT EXISTS cache(id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT unique, value TEXT)`)

	if _err != nil {
		fmt.Println(_err)
	}

	res, _err := stmt.Exec()
	if _err != nil {
		fmt.Println(_err)
	}
	fmt.Println(res)

	fmt.Println(">> StorageQueue ... Run")

	for {

		item := <-q.que

		fmt.Println(">> QUEUE", item)

		var id int
		var value string
		hasRow := false

		rows, _err := db.Query("SELECT id, value FROM cache WHERE key=?", item.key)
		if _err != nil {
			fmt.Println(_err)
		}

			
		for rows.Next() {
			
			_err = rows.Scan(&id, &value)
			fmt.Println("GET key=", item.key, "id=", id, "value=", value)
			hasRow = true

			break
		}

		rows.Close()

		switch item.itemType {
		case STORAGE_GET:
			fmt.Println("STORAGE_GET", value)
			item.value = value
			break
		case STORAGE_SET:
			if hasRow {
				fmt.Println("UPDATE")
				ExecSQL(db, "UPDATE `cache` SET `value`=? WHERE id=?", item.value, id)

			} else {
				fmt.Println("INSERT")
				ExecSQL(db, "INSERT INTO `cache`(`key`, `value`) VALUES(?, ?)", item.key, item.value)

			}			
			break
		}

		item.evt <- 0	

	}	
}


func ExecSQL(db *sql.DB, sqlText string, args ...interface{}) {
	
	stmt, _err := db.Prepare(sqlText)
	if _err != nil {
		fmt.Println("ExecSQL ERROR", _err)
	} else {

		res, _err := stmt.Exec(args...)

		if _err != nil {
			fmt.Println("ExecSQL ERROR", _err)
		} else {

			id, _err := res.LastInsertId()
			if _err == nil {
				fmt.Println("ExecSQL SUCCESS!", "LastInsertId", id)
			}
		}
	}
}



func (handler *ConnectionHandler) PacketReceived(packet []interface{}) {

	fmt.Println("PacketReceived", packet)

	cmdName := packet[0]
	key := packet[1].(string)

	switch cmdName {
	case "SET":
		handler.que.Set(key, packet[2].(string))
		break
	case "GET":
		value := handler.que.Get(key)
		(*handler.conn).Write(simple_cache.PackArrayPacket(value))
		break
	}
}

func (handler *ConnectionHandler) Run() {

	fmt.Println("handler >> run..........")

	receiver := simple_cache.DataReceiver{}
	
	recvBuf := make([]byte, 4096)

	for {

		nRecv, _err :=  (*handler.conn).Read(recvBuf)
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
			handler.PacketReceived(packet)

		}

	}

}


func main() {

	port := 16384
	if len(os.Args) > 1 {
		port, _ = strconv.Atoi(os.Args[1])
	}

	fmt.Printf("starting simple redis server port=%v\n", port)

	listener, _err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if _err != nil {
		fmt.Println(_err)
		os.Exit(1)
	}

	que := NewStorageQueue()
	go que.Run()


	for {
		conn, _err := listener.Accept()
		if _err != nil {
			fmt.Println(_err)
			os.Exit(1)
		}

		handler := ConnectionHandler{conn: &conn, que: que}

		go handler.Run()

	}

}
