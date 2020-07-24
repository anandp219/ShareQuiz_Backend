package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sharequiz/app/database"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

// GameRoom GameRoom data
type GameRoom struct {
	Language Language `json:"language,string"`
	Topic    Topic    `json:"topic,string"`
}

//CreateRoom create room for a game for other players to join.
func CreateRoom(c *gin.Context) {
	phoneNumber := c.Query("phone_number")
	room := c.Query("room")
	gameRoom := GameRoom{}
	err := json.Unmarshal([]byte(room), &gameRoom)
	if err != nil {
		sendError(c, "check the topic and room")
		return
	}
	err = ValidatePhoneNumber(phoneNumber)
	if err != nil {
		sendError(c, "check the phone number")
		return
	}
	fmt.Println("creating room for the phone number " + phoneNumber)
	roomID := 0
	lastRoomID, err := database.RedisClient.Get(LastRoomIDKey).Result()
	if err == redis.Nil {
		roomID = 1
	} else if err != nil {
		sendError(c, "error while creating game "+err.Error())
		return
	} else {
		roomID, _ = strconv.Atoi(lastRoomID)
		roomID++
	}
	gameRoomStr, err := json.Marshal(gameRoom)
	if err != nil {
		sendError(c, "error while creating room")
		return
	}
	_, err = database.RedisClient.Set(LastRoomIDKey, roomID, 0).Result()
	if err != nil {
		sendError(c, "error while creating room")
		return
	}
	_, err = database.RedisClient.Set("room-"+strconv.Itoa(roomID), gameRoomStr, 0).Result()
	if err != nil {
		sendError(c, "error while creating room")
		return
	}
	sendSuccess(c, strconv.Itoa(roomID))
}

//JoinRoom join room for a game.
func JoinRoom(c *gin.Context) {
	phoneNumber := c.Query("phone_number")
	roomID := c.Query("roomID")
	roomData := c.Query("room")
	gameRoom := GameRoom{}

	err := json.Unmarshal([]byte(roomData), &gameRoom)
	if err != nil {
		sendError(c, "check the topic and room")
		return
	}
	err = ValidatePhoneNumber(phoneNumber)
	if err != nil {
		sendError(c, "check the phone number")
		return
	}
	fmt.Println("joining room " + roomID + " for the phone number " + phoneNumber)
	savedRoomData, err := database.RedisClient.Get("room-" + roomID).Result()
	if err != nil {
		sendError(c, "error while joining game "+err.Error())
		return
	}
	savedGameRoom := GameRoom{}
	err = json.Unmarshal([]byte(savedRoomData), &savedGameRoom)
	if err != nil {
		sendError(c, "error while joining game "+err.Error())
		return
	}

	if savedGameRoom.Language != gameRoom.Language ||
		savedGameRoom.Topic != gameRoom.Topic {
		sendError(c, "the topic and language should be exact for game "+err.Error())
		return
	}
	sendSuccess(c, roomID)
}

func sendError(c *gin.Context, errorString string) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"message": errorString,
	})
}

func sendSuccess(c *gin.Context, roomID string) {
	c.JSON(http.StatusOK, gin.H{
		"roomID": roomID,
	})
}
