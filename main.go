package main

import (
	"fmt"
	"net/http"
	"os"
	"sharequiz/app"
	"sharequiz/app/admin"
	"sharequiz/app/database"
	"sharequiz/app/socket"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.Use(static.Serve("/static/images", static.LocalFile("./app/static/images", false)))
	router.GET("/ping", pong)
	v1 := router.Group("/api/v1")
	{
		v1.GET("/otp", app.GetOTP)
		v1.PUT("/otp", app.VerifyOTP)
	}
	v2 := router.Group("/api/admin")
	{
		v2.GET("/questions", admin.GetQuestions)
		v2.GET("/game", admin.GetGame)
		v2.GET("/otp", admin.GetOtp)
		v2.GET("/create_game", admin.CreateGame)
	}
	database.InitRedis()
	database.InitElastic()
	go socket.InitPlayerJoinSocket()
	go socket.InitGameSocket()
	err := router.Run(os.Getenv("PORT"))
	if err != nil {
		fmt.Println(err)
	}
}

func pong(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
	return
}
