package admin

import (
	"encoding/json"
	"net/http"
	"sharequiz/app"
	"sharequiz/app/database"

	"github.com/gin-gonic/gin"
)

//GetQuestions admin function for getting the question
func GetQuestions(c *gin.Context) {
	questions, err := app.GetGameQuestions(app.India, app.English, app.NumOfQuestionsInGame)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"questions": make([]app.Question, 0),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"questions": questions,
		})
	}
}

//GetGame get game object at any instant
func GetGame(c *gin.Context) {
	gameID := c.Query("game_id")
	gameData, err := database.RedisClient.Get(gameID).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"game": app.Game{},
		})
		return
	}
	game := &app.Game{}
	err = json.Unmarshal([]byte(gameData), game)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"game": app.Game{},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"game": game,
	})
}
