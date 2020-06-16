package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"sharequiz/app"
	"sharequiz/app/database"
	"time"

	"github.com/go-redis/redis"

	gosocketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
)

//Room to joined by the clients
type Room struct {
	Room        string `json:"room"`
	PhoneNumber string `json:"phoneNumber"`
}

var server *gosocketio.Server
var clientToRoomMap = make(map[string]string)

// InitGameSocket is used to initialise the socket.
func InitGameSocket() {
	server = gosocketio.NewServer(transport.GetDefaultWebsocketTransport())
	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		log.Println("Connected")
	})

	server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		go disconnectPlayer(c)
	})

	server.On("/join", func(c *gosocketio.Channel, room Room) {
		go playerJoin(c, room)
	})

	server.On("/answer", func(c *gosocketio.Channel, gameString string) {
		go answerQuestion(c, gameString)
	})

	server.On("/time_out", func(c *gosocketio.Channel, gameString string) {
		go handleTimeout(c, gameString)
	})

	serveMux := http.NewServeMux()
	serveMux.Handle("/socket.io/", server)

	log.Println("Starting server...")
	log.Panic(http.ListenAndServe(":8082", serveMux))
}

func disconnectPlayer(c *gosocketio.Channel) {
	errorMessage := "Error while disconnecting player"
	defer handleError(c)
	room := clientToRoomMap[c.Id()]
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
	delete(clientToRoomMap, c.Id())
	server.BroadcastTo(room, "/disconnect", string(gameJSON))
}

func playerJoin(c *gosocketio.Channel, room Room) {
	errorMessage := "error while getting game for the gameId"
	defer handleError(c)
	roomString := string(room.Room)
	c.Join(roomString)
	clientToRoomMap[c.Id()] = roomString
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

func handleTimeout(c *gosocketio.Channel, gameString string) {
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

func answerQuestion(c *gosocketio.Channel, gameString string) {
	errorMessage := "error while answering the question"
	game := &app.Game{}
	err := json.Unmarshal([]byte(gameString), game)
	if err != nil {
		panic(errorMessage)
	}
	questionNumber := game.QuestionNumber
	question := game.Questions[questionNumber]
	totalAnswered := 0
	for key, value := range question.PlayerAnswers {
		if value == question.Answer {
			game.Scores[key][game.QuestionNumber] = 10
		} else {
			game.Scores[key][game.QuestionNumber] = 0
		}
		totalAnswered++
	}
	sendNewQuestion(totalAnswered, game, errorMessage)
}

func sendNewQuestion(totalAnswered int, game *app.Game, errorMessage string) {
	event := "/new_answer"
	if totalAnswered == game.NumberOfPlayers {
		if game.QuestionNumber == game.MaxQuestions {
			game.Status = app.Finished
			event = "/game_over"
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
	server.BroadcastTo(game.ID, event, string(gameJSON))
	if event == "/new_answer" && totalAnswered == game.NumberOfPlayers {
		go sendQuestion(game.ID, game.QuestionNumber)
	}
}

func sendQuestion(room string, questionNumber int) {
	time.Sleep(2 * time.Second)
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
	server.BroadcastTo(room, "/new_question", string(gameJSON))
}

func handleError(c *gosocketio.Channel) {
	if r := recover(); r != nil {
		log.Println(r)
		log.Println("error while joining room for the sockets")
		c.Close()
	}
}
