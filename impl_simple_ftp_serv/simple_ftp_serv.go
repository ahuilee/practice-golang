/*
	https://tools.ietf.org/html/rfc959

	golang simple ftp server implementation
	2017.7.16 by ahui
	learn goals: basic syntax, file read write, socket, thread
*/

package main

import (
	"os"
	"fmt"
	"log"
	"net"
	"strings"
	"bufio"
	"strconv"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	//"unicode/utf8"
)

const STATE_UNAUTH int = 0
const STATE_AUTHED int = 1

const COMMAND_USER string = "USER"
const COMMAND_PASS string = "PASS"
const COMMAND_SYST string = "SYST"
const COMMAND_FEAT string = "FEAT"
const COMMAND_PWD string = "PWD"
const COMMAND_TYPE string = "TYPE"
const COMMAND_PASV string = "PASV"
const COMMAND_LIST string = "LIST"
const COMMAND_CDUP string = "CDUP"
const COMMAND_CWD string = "CWD"
const COMMAND_RETR string = "RETR"
const COMMAND_STOR string = "STOR"

const RESPONSE_TYPE_SET_OK = "200"
const RESPONSE_PWD = "211"
const RESPONSE_NAME_SYS_TYPE = "215"
const RESPONSE_WELCOME_MSG string = "220"
const RESPONSE_ENTERING_PASV_MODE = "227"
const RESPONSE_LOGIN_SUCCESS string = "230"
const RESPONSE_OK = "250"
const RESPONSE_PASSWORD_REQUIRED string = "331"


type FTPConn struct {
	conn net.Conn
	dtpListener net.Listener
	dtpConn net.Conn
}


func main() {

	dirname := os.Args[1]
	rootDir, _err := filepath.Abs(dirname)
	if _err != nil {
		fmt.Println(_err)
		os.Exit(1)
	}

	fmt.Printf("root=%s\n", rootDir)

	listenPort := 16384

	fmt.Printf("listen simple ftp server...%d", listenPort)

	servSock := listenTCP(listenPort)
	
	for {
		conn, err := servSock.Accept()
		if err != nil {
			os.Exit(1)
		}

		log.Printf("conn %s", conn)

		ftpConn := FTPConn{conn: conn}

		go handleConn(&ftpConn, rootDir)

	}
}


func listenTCP(port int) net.Listener {

	fmt.Printf("listen :%d\n",  port)

	listener, _err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if _err != nil {
		//fmt.Println(_err.Error())
		os.Exit(1)
	}

	return listener
}


func reply(ftpConn *FTPConn, code string, text string) {
	msg := fmt.Sprintf("%s %s\n", code, text)
	log.Printf("send: %s\n", msg)
	ftpConn.conn.Write([]byte(msg))
}

func recv(reader *bufio.Reader)(string, string) {
	recvData, err := reader.ReadString('\n')

	if err != nil {
		return "", ""
	}

	msg := strings.TrimSpace(recvData)

	log.Printf("recv=[%s]\n", msg)

	idx := strings.Index(msg, " ")

	opCode := msg
	var arg string

	if idx > 0 {
		opCode = msg[:idx]
		arg = msg[idx+1:]
	}

	//arg = strings.TrimRight(arg, "\n")

	log.Printf("recv opCode=[%s] arg=[%s]", opCode, arg)

	return opCode, arg
}

func encodeHostPort(host string, port int)(string) {
	numbers := strings.Split(host, ".")

	numbers = append(numbers, strconv.Itoa(port >> 8))
	numbers = append(numbers, strconv.Itoa(port % 256))


	return strings.Join(numbers, ",")
}

func getConnHostPort(conn net.Conn)(string, int) {
	ipaddr := conn.RemoteAddr().String()
	spIdx := strings.Index(ipaddr, ":")

	host := ipaddr[:spIdx]
	port_i, _ := strconv.Atoi(ipaddr[spIdx+1:len(ipaddr)])

	return host, port_i

}




func handlePASV(ftpConn *FTPConn) {


	fmt.Printf("handlePASV\n")
	dtpConn, _ := ftpConn.dtpListener.Accept()
	fmt.Printf("handlePASV Accept\n")

	ftpConn.dtpConn = dtpConn

}

func ftpList(ftpConn *FTPConn, dirPath string) {
		reply(ftpConn, "150", "list test")

		infos, _ := ioutil.ReadDir(dirPath)

		dtpConn, _ := ftpConn.dtpListener.Accept()

		for _, info := range infos {


			permIsDir := "-"
			permIsDirExec := "-"

			if info.IsDir() {
				permIsDir = "d"
				permIsDirExec = "x"
			} else {
				permIsDir = "-"
				permIsDirExec = "-"
			}

			filePerm := strings.Join([]string{permIsDir, "r", "w", permIsDirExec, "r", "-", "-", "r", "-", "-"}, "")

			fileSizeStr := fmt.Sprintf("%v", info.Size())
			paddingSpaceCount := 14 - len(fileSizeStr)
			fileSizeStr = strings.Repeat(" ", paddingSpaceCount) + fileSizeStr

			fmt.Printf("padding space=%v fileSizeStr=[%v]\n", paddingSpaceCount, fileSizeStr)

			fileName := info.Name()

			packet := []string{filePerm, "1", "ftp", "ftp", fileSizeStr, "May 02 2017", fileName}

			sendData := fmt.Sprintf("%s\n", strings.Join(packet, " "))
			fmt.Printf("DTP send: [%s]\n", sendData)
			dtpConn.Write([]byte(sendData))

		}

		dtpConn.Close()
		reply(ftpConn, "226", "Directory send OK.")

}

func ftpRETR(ftpConn *FTPConn, fullPath string) {

	fmt.Printf("ftpRETR >> %v\n", fullPath)

	dtpConn, _ := ftpConn.dtpListener.Accept()

	bytes, _ := ioutil.ReadFile(fullPath)

	dtpConn.Write(bytes)
	dtpConn.Close()

	reply(ftpConn, RESPONSE_OK, "OK")

}

func ftpSTOR(ftpConn *FTPConn, fullPath string) {

	fmt.Printf("ftpSTOR >> %v\n", fullPath)

	dtpConn, _ := ftpConn.dtpListener.Accept()

	buf := make([]byte, 4096)

	f, _err := os.Create(fullPath)

	if _err != nil {

		return
	}

	for {
		nRecv, _err := dtpConn.Read(buf)
		if _err != nil {
			if _err == io.EOF {
				fmt.Println("ftpSTOR >> EOF")
				break
			}
		}
		fmt.Printf("recv=%v\n", nRecv)
		f.Write(buf[:nRecv])
	}

	f.Close()
	dtpConn.Close()

	reply(ftpConn, RESPONSE_OK, "OK")

}

func handleConn(ftpConn *FTPConn, rootDir string) {

	log.Printf("handleConn")

	rootDir, _err := filepath.Abs(rootDir)
	if _err != nil {
		fmt.Println(_err)
	}
	fmt.Printf("handleConn >> %v \n", rootDir)

	reader := bufio.NewReader(ftpConn.conn)
	state := STATE_UNAUTH

	reply(ftpConn, RESPONSE_WELCOME_MSG, "ahui Simple FTP Server")

	curDir := rootDir

	for {

		opCode, arg := recv(reader)


		if state == STATE_UNAUTH {

			if opCode == COMMAND_USER {
				reply(ftpConn, RESPONSE_PASSWORD_REQUIRED, fmt.Sprintf("Password required to access user %s", arg))

				opCode, arg := recv(reader)

				if opCode == COMMAND_PASS {
					if arg == "123" {

						state = STATE_AUTHED

						reply(ftpConn, RESPONSE_LOGIN_SUCCESS, "Logged in")

					}

				}

			}

		} else {

			switch opCode {
			case COMMAND_SYST:
				reply(ftpConn, RESPONSE_NAME_SYS_TYPE, "UNIX Type: L8")
				break
			case COMMAND_FEAT:
				reply(ftpConn, "211", "END")
				break
			case COMMAND_PWD:
				reply(ftpConn, RESPONSE_PWD, curDir)
				break
			case COMMAND_TYPE:
				if arg == "I" {
					reply(ftpConn, RESPONSE_TYPE_SET_OK, fmt.Sprintf("Type set to %s", arg))
				}
				break
			case COMMAND_PASV:
				dtpListener := listenTCP(0)
				
				ftpConn.dtpListener = dtpListener
				addr := dtpListener.Addr().(*net.TCPAddr)
				host := "127.0.0.1"
				fmt.Printf("PASV host=%s port=%d\n", host, addr.Port)

				reply(ftpConn, RESPONSE_ENTERING_PASV_MODE, encodeHostPort(host, addr.Port))
				break
			case COMMAND_LIST:
				ftpList(ftpConn, curDir)
				break
			case COMMAND_CWD:
				curDir = arg
				fmt.Printf("CWD curDir=%v\n", curDir)

				reply(ftpConn, RESPONSE_OK, "OK")
				break
			case COMMAND_CDUP:
				fmt.Printf("curDir=%v\n", curDir)
				curDir = path.Dir(curDir)
				if curDir == "." {
					curDir = rootDir
				}
				fmt.Printf("CD curDir=%v\n", curDir)

				reply(ftpConn, RESPONSE_OK, "OK")
				break
			case COMMAND_RETR:
				fmt.Printf("RETR curDir=%v\n", curDir)
				ftpRETR(ftpConn, path.Join(curDir, arg))
				break
			case COMMAND_STOR:
				//fileNameRunes := []rune(arg)
				//fileName := string([]byte(string(fileNameRunes)))
				fileName := arg
				fmt.Printf(">> fileName =%v\n", fileName)

				ftpSTOR(ftpConn, path.Join(curDir, fileName))

				break


			}
		}
	}
}


