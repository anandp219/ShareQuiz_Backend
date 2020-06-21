package socket

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sharequiz/app"
	"strconv"
)

// WaitingSockets variable is used for connection.
var WaitingSockets map[string][]net.Conn

// Init is used to initialise the socket.
func Init() {
	service := ":8081"
	WaitingSockets = make(map[string][]net.Conn)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer closeSocket(conn)
	topicID, language := getClientData(conn)
	key := topicID + "_" + language
	if len(WaitingSockets[key]) == 0 {
		WaitingSockets[key] = append(WaitingSockets[key], conn)
	} else {
		socketsForTopic := WaitingSockets[key]
		secondConn := socketsForTopic[0]
		// _, err := strconv.Atoi(language)
		// _, err = strconv.Atoi(topicID)
		languageEnum, err := strconv.Atoi(language)
		topicEnum, err := strconv.Atoi(topicID)
		gameID, err := app.CreateGame(app.NumOfQuestionsInGame, app.Language(languageEnum), 2, app.Topic(topicEnum))
		// gameID := "1"
		if err != nil {
			panic("Socket Error")
		}
		WaitingSockets[key] = socketsForTopic[1:]
		_, err = conn.Write([]byte(gameID))
		_, err = secondConn.Write([]byte(gameID))
		conn.Close()
		secondConn.Close()
	}
}

func getClientData(conn net.Conn) (string, string) {
	buffer := make([]byte, 0, 4096)
	tmp := make([]byte, 256)
	n, err := conn.Read(tmp)
	buffer = append(buffer, tmp[:n]...)
	m := make(map[string]string)
	err = json.Unmarshal(buffer, &m)
	if err != nil {
		panic("socket error")
	}
	return m["topicId"], m["language"]
}

func closeSocket(conn net.Conn) {
	if r := recover(); r != nil {
		_, _ = conn.Write([]byte("-1"))
		conn.Close()
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Printf("Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
