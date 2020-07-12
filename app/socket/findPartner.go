package socket

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sharequiz/app"
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

var playerJoinServer *socketio.Server

// GameData Initial game data of the game
type GameData struct {
	Topic    app.Topic    `json:"topic,string"`
	Language app.Language `json:"language,string"`
}

// WaitingSockets variable is used for connection.
var WaitingSockets = make(map[string][]socketio.Conn)

// SocketToTopicMap is a map from the socket id to the game topic
var SocketToTopicMap = make(map[string]string)

// RoomToLock Room locks for synchronization of go routines for a room.
var RoomToLock = make(map[string]*sync.Mutex)

// TopicToLock Topic locks for synchronization of go routines for a topic.
var TopicToLock = make(map[string]*sync.Mutex)

// InitPlayerJoinSocket is used to initialise the socket.
func InitPlayerJoinSocket() {
	playerJoinServer, err = socketio.NewServer(nil)
	if err != nil {
		panic(err)
	}
	playerJoinServer.OnConnect("/", func(s socketio.Conn) error {
		log.Println("Connected")
		return nil
	})

	playerJoinServer.OnEvent("/", "join", func(c socketio.Conn, gameData GameData) {
		go connectJoin(c, gameData)
	})

	playerJoinServer.OnDisconnect("/", func(s socketio.Conn, reason string) {
		go disconnectJoin(s)
	})

	go playerJoinServer.Serve()
	defer playerJoinServer.Close()

	http.Handle("/socket.io/join_game/", playerJoinServer)
	log.Println("Serving at localhost" + os.Getenv("PARTNER_PORT"))
	log.Fatal(http.ListenAndServe(os.Getenv("PARTNER_PORT"), nil))
}

func connectJoin(conn socketio.Conn, gameData GameData) {
	fmt.Println("connectjoin")
	key := gameData.Topic.String() + "_" + gameData.Language.String()
	lockTopic(key)
	SocketToTopicMap[conn.ID()] = key
	defer handleConnectJoinError(conn, key)
	if len(WaitingSockets[key]) == 0 {
		WaitingSockets[key] = append(WaitingSockets[key], conn)
	} else {
		socketsForTopic := WaitingSockets[key]
		secondConn := socketsForTopic[0]
		gameID, err := app.CreateGame(app.NumOfQuestionsInGame, gameData.Language, 2, gameData.Topic)
		if err != nil {
			fmt.Println("error for game is " + err.Error())
			panic("Socket Error")
		}
		RoomToLock[gameID] = &sync.Mutex{}
		conn.Emit("game", gameID)
		secondConn.Emit("game", gameID)
		WaitingSockets[key] = socketsForTopic[1:]
	}
	unlockTopic(key)
}

func disconnectJoin(conn socketio.Conn) {
	key, ok := SocketToTopicMap[conn.ID()]
	if !ok {
		return
	}
	lockTopic(key)
	defer handleDisconnectJoinError(conn, key)
	delete(SocketToTopicMap, conn.ID())
	socketsForTopic, ok := WaitingSockets[key]
	if ok {
		for i, value := range socketsForTopic {
			if value.ID() == conn.ID() {
				WaitingSockets[key] = append(socketsForTopic[:i], socketsForTopic[i+1:]...)
			}
		}
	}
	unlockTopic(key)
}

func handleConnectJoinError(conn socketio.Conn, key string) {
	if r := recover(); r != nil {
		unlockTopic(key)
		conn.Close()
	}
}

func handleDisconnectJoinError(conn socketio.Conn, key string) {
	if r := recover(); r != nil {
		unlockTopic(key)
	}
}

func lockTopic(key string) {
	if mutex, ok := TopicToLock[key]; ok {
		mutex.Lock()
	} else {
		TopicToLock[key] = &sync.Mutex{}
		TopicToLock[key].Lock()
	}
}

func unlockTopic(key string) {
	if mutex, ok := TopicToLock[key]; ok {
		mutex.Unlock()
	}
}
