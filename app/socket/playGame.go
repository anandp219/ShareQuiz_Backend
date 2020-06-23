package socket

import (
	"encoding/json"
	"log"
	"net/http"
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
		log.Println("Connected")
		return nil
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		go disconnectPlayer(s)
	})

	server.OnEvent("/", "message", func(c socketio.Conn, room Room) {
		log.Println("message")
	})

	server.OnEvent("/", "join", func(c socketio.Conn, room Room) {
		go playerJoin(c, room)
	})

	server.OnEvent("/", "answer", func(c socketio.Conn, gameString string) {
		go answerQuestion(c, gameString)
	})

	server.OnEvent("/", "time_out", func(c socketio.Conn, gameString string) {
		go handleTimeout(c, gameString)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	log.Println("Serving at localhost:8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func disconnectPlayer(c socketio.Conn) {
	errorMessage := "Error while disconnecting player"
	defer handleDisconnectError(c)
	room, ok := clientToRoomMap[c.ID()]
	if !ok {
		return
	}
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

func playerJoin(c socketio.Conn, room Room) {
	errorMessage := "error while getting game for the gameId"
	defer handleError(c)
	roomString := string(room.Room)
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
	if len(game.Players) == 2 {
		go sendQuestion(roomString, 1)
	}
}

func handleTimeout(c socketio.Conn, gameString string) {
	errorMessage := "error while handling the timeout event"
	game := &app.Game{}
	err := json.Unmarshal([]byte(gameString), game)
	if err != nil {
		panic(errorMessage)
	}
	questionNumber := game.QuestionNumber
	question := game.Questions[questionNumber]
	totalAnswered := 0
	for range question.PlayerAnswers {
		totalAnswered++
	}
	sendNewQuestion(totalAnswered, game, errorMessage)
}

func answerQuestion(c socketio.Conn, gameString string) {
	errorMessage := "error while answering the question"
	game := &app.Game{}
	err := json.Unmarshal([]byte(gameString), game)
	if err != nil {
		panic(errorMessage)
	}
	questionNumber := game.QuestionNumber
	question := game.Questions[questionNumber]
	totalAnswered := 0
	for range question.PlayerAnswers {
		totalAnswered++
	}
	sendNewQuestion(totalAnswered, game, errorMessage)
}

func sendNewQuestion(totalAnswered int, game *app.Game, errorMessage string) {
	event := "new_answer"
	if totalAnswered == game.NumberOfPlayers {
		if game.QuestionNumber == game.MaxQuestions {
			game.Status = app.Finished
			event = "game_over"
		} else {
			game.QuestionNumber++
		}
	}
	gameJSON, err := json.Marshal(game)
	if err != nil {
		panic(errorMessage)
	}
	_, err = database.RedisClient.Set(game.ID, string(gameJSON), 0).Result()
	if err != nil {
		panic(errorMessage)
	}
	server.BroadcastToRoom("/", game.ID, event, string(gameJSON))
	if event == "new_answer" && totalAnswered == game.NumberOfPlayers {
		go sendQuestion(game.ID, game.QuestionNumber)
	}
}

func sendQuestion(room string, questionNumber int) {
	time.Sleep(5 * time.Second)
	errorMessage := "error while sending question to the users"
	gameData, err := database.RedisClient.Get(room).Result()
	if err != nil {
		panic(errorMessage)
	}
	game := &app.Game{}
	err = json.Unmarshal([]byte(gameData), game)
	if err != nil {
		panic(errorMessage)
	}
	game.QuestionNumber = questionNumber
	gameJSON, err := json.Marshal(game)
	_, err = database.RedisClient.Set(room, string(gameJSON), 0).Result()
	if err != nil {
		panic(errorMessage)
	}
	server.BroadcastToRoom("/", room, "new_question", string(gameJSON))
}

func handleError(c socketio.Conn) {
	if r := recover(); r != nil {
		log.Println(r)
		log.Println("error while joining room for the sockets")
		c.Close()
	}
}

func handlePlayerJoinError(c socketio.Conn) {
	if r := recover(); r != nil {
		log.Println(r)
		c.Close()
	}
}

func handleDisconnectError(c socketio.Conn) {
	if r := recover(); r != nil {
		log.Println(r)
		log.Println("error while removing the socket from the room")
	}
}
