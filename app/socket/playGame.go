package socket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sharequiz/app"
	"sharequiz/app/database"
	"time"

	"github.com/go-redis/redis"
	socketio "github.com/googollee/go-socket.io"
)

//Room to joined by the clients
type Room struct {
	Room        string `json:"room"`
	PhoneNumber string `json:"phoneNumber"`
}

var server *socketio.Server
var clientToRoomMap = make(map[string]string)
var roomToClientID = make(map[string][]string)
var err error

// InitGameSocket is used to initialise the socket.
func InitGameSocket() {
	server, err = socketio.NewServer(nil)
	if err != nil {
		panic(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		log.Println("connect game")
		return nil
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("disconnect game")
		go disconnectPlayer(s)
	})

	server.OnEvent("/", "join", func(c socketio.Conn, room Room) {
		log.Println("join game")
		go playerJoin(c, room)
	})

	server.OnEvent("/", "answer", func(c socketio.Conn, gameString string) {
		log.Println("answer")
		go answerQuestion(c, gameString)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	log.Println("Serving at localhost" + os.Getenv("GAME_PORT"))
	log.Fatal(http.ListenAndServe(os.Getenv("GAME_PORT"), nil))
}

func disconnectPlayer(c socketio.Conn) {
	errorMessage := "Error while disconnecting player"
	room, ok := clientToRoomMap[c.ID()]
	if !ok {
		return
	}
	lockRoom(room)
	defer handleDisconnectError(c, room)
	clientIDsForRoom := roomToClientID[room]
	for _, clientID := range clientIDsForRoom {
		delete(clientToRoomMap, clientID)
	}
	delete(roomToClientID, room)
	gameData, err := database.RedisClient.Get(room).Result()
	if err != nil || err == redis.Nil {
		log.Println(err)
		panic(errorMessage)
	}
	game := &app.Game{}
	err = json.Unmarshal([]byte(gameData), game)
	if game.Status != app.Finished {
		if err != nil {
			panic(errorMessage)
		}
		game.Status = app.Disconnected
		gameJSON, err := json.Marshal(game)
		_, err = database.RedisClient.Set(room, string(gameJSON), 0).Result()
		if err != nil {
			panic(errorMessage)
		}
		server.BroadcastToRoom("/", room, "disconnect", string(gameJSON))
	}
	unlockRoom(room)
}

func playerJoin(c socketio.Conn, room Room) {
	errorMessage := "error while joining player for the game"
	roomString := string(room.Room)
	lockRoom(roomString)
	defer handlePlayerJoinError(c, roomString)
	c.Join(roomString)
	clientToRoomMap[c.ID()] = roomString
	clientIds, ok := roomToClientID[roomString]
	if !ok {
		clientIds = make([]string, 0)
	}
	roomToClientID[roomString] = append(clientIds, c.ID())
	gameData, err := database.RedisClient.Get(roomString).Result()
	if err != nil || err == redis.Nil {
		log.Println(err)
		panic(errorMessage)
	}
	game := &app.Game{}
	err = json.Unmarshal([]byte(gameData), game)
	if err != nil {
		panic(errorMessage)
	}
	//1 is added as to use it as 1 indexed
	scores := make([]int, game.MaxQuestions+1)
	if _, ok := game.Players[room.PhoneNumber]; !ok {
		game.Players[room.PhoneNumber] = app.Player{
			ID:       room.PhoneNumber,
			Score:    0,
			Selected: 0,
		}
		game.Scores[room.PhoneNumber] = scores
	}
	gameJSON, err := json.Marshal(game)
	_, err = database.RedisClient.Set(roomString, string(gameJSON), 0).Result()
	if err != nil {
		panic(errorMessage)
	}
	unlockRoom(roomString)
	if len(game.Players) == 2 {
		go sendNewQuestion(game, true, c)
	}
}

func answerQuestion(c socketio.Conn, gameString string) {
	errorMessage := "error while handling the timeout event"
	game := &app.Game{}
	err := json.Unmarshal([]byte(gameString), game)
	if err != nil {
		panic(errorMessage)
	}
	lockRoom(game.ID)
	defer handleAnswerQuestionError(c, game.ID)
	gameData, err := database.RedisClient.Get(game.ID).Result()
	if err != nil || err == redis.Nil {
		panic(errorMessage)
	}
	oldGame := &app.Game{}
	err = json.Unmarshal([]byte(gameData), oldGame)
	if err != nil {
		panic(errorMessage)
	}
	for key, value := range game.Questions[game.QuestionNumber].PlayerAnswers {
		oldGame.Questions[oldGame.QuestionNumber].PlayerAnswers[key] = value
		oldGame.Scores[key] = game.Scores[key]
	}
	gameJSON, err := json.Marshal(oldGame)
	if err != nil {
		panic(errorMessage)
	}
	_, err = database.RedisClient.Set(game.ID, string(gameJSON), 0).Result()
	if err != nil {
		panic(errorMessage)
	}
	server.BroadcastToRoom("/", game.ID, "new_answer", string(gameJSON))
	sendNewQuestion(oldGame, false, c)
}

func sendNewQuestion(game *app.Game, shouldLockRoom bool, c socketio.Conn) {
	if shouldLockRoom {
		lockRoom(game.ID)
		defer handleSendNewQuestionError(c, game.ID)
	}
	errorMessage := "error while sending a new question for game "
	fmt.Println("send new question:", game.QuestionNumber, (game.Questions[game.QuestionNumber].PlayerAnswers))
	event := "new_question"
	questionNumber := game.QuestionNumber
	question := game.Questions[questionNumber]
	totalAnswered := 0
	for range question.PlayerAnswers {
		totalAnswered++
	}
	if totalAnswered == game.NumberOfPlayers || questionNumber == 0 {
		if game.QuestionNumber == game.MaxQuestions {
			game.Status = app.Finished
			event = "game_over"
		} else {
			game.QuestionNumber++
			time.Sleep(2 * time.Second)
		}
		gameJSON, err := json.Marshal(game)
		if err != nil {
			panic(errorMessage)
		}
		_, err = database.RedisClient.Set(game.ID, string(gameJSON), 0).Result()
		if err != nil {
			panic(errorMessage)
		}
		fmt.Println("Sending new question" + game.ID)
		server.BroadcastToRoom("/", game.ID, event, string(gameJSON))
	}
	unlockRoom(game.ID)
	if event == "game_over" {
		deleteLockRoom(game.ID)
	}
}

func handleSendNewQuestionError(c socketio.Conn, room string) {
	if r := recover(); r != nil {
		unlockRoom(room)
		c.Close()
	}
}

func handleAnswerQuestionError(c socketio.Conn, room string) {
	if r := recover(); r != nil {
		unlockRoom(room)
		c.Close()
	}
}

func handlePlayerJoinError(c socketio.Conn, room string) {
	if r := recover(); r != nil {
		unlockRoom(room)
		c.Close()
	}
}

func handleDisconnectError(c socketio.Conn, room string) {
	if r := recover(); r != nil {
		unlockRoom(room)
	}
}

func lockRoom(roomID string) {
	if mutex, ok := RoomToLock[roomID]; ok {
		mutex.Lock()
	}
}

func unlockRoom(roomID string) {
	if mutex, ok := RoomToLock[roomID]; ok {
		mutex.Unlock()
	}
}

func deleteLockRoom(roomID string) {
	delete(RoomToLock, roomID)
}
