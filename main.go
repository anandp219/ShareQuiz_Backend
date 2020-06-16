package main

import (
	"fmt"
	"net/http"
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
	}
	database.InitRedis()
	database.InitElastic()
	go socket.Init()
	go socket.InitGameSocket()
	err := router.Run()
	if err != nil {
		fmt.Println("Error while starting server")
	}
}

func pong(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
	return
}
