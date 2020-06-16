package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"sharequiz/app/database"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

// PhoneVerificationData PhoneVerificationData
type PhoneVerificationData struct {
	PhoneNumber           string    `json:"phoneNumber"`
	IsVerified            bool      `json:"isVerified"`
	Otp                   string    `json:"otp"`
	VerificationTimestamp time.Time `json:"verificationTimestamp"`
	SentTimestamp         time.Time `json:"sentTimestamp"`
}

// GetOTP GetOTP
func GetOTP(c *gin.Context) {
	phoneNumber := c.Query("phone_number")
	err := validatePhoneNumber(phoneNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "check the phone number",
		})
		return
	}
	val, err := database.RedisClient.Get(phoneNumber).Result()
	if err == redis.Nil {
		otp := rangeIn(1000, 9999)
		data := PhoneVerificationData{
			phoneNumber,
			false,
			otp,
			time.Time{},
			time.Now(),
		}
		dataStr, err := json.Marshal(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error while sending OTP.",
			})
			return
		}
		_, err = database.RedisClient.Set(phoneNumber, string(dataStr), 0).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error while sending OTP.",
			})
			return
		}
		fmt.Println("Otp send for phoneNumber " + phoneNumber + " is : " + otp)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error while sending OTP.",
		})
		return
	} else {
		data := &PhoneVerificationData{}
		err := json.Unmarshal([]byte(val), data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error while sending OTP.",
			})
			return
		}
		otp := rangeIn(1000, 9999)
		newData := PhoneVerificationData{
			phoneNumber,
			false,
			otp,
			time.Time{},
			time.Now(),
		}
		dataStr, err := json.Marshal(newData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error while sending OTP.",
			})
			return
		}
		_, err = database.RedisClient.Set(phoneNumber, string(dataStr), 0).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error while sending OTP.",
			})
			return
		}
		fmt.Println("Otp send for phoneNumber " + phoneNumber + " is : " + otp)
	}
	c.JSON(http.StatusOK, gin.H{
		"result": "success",
	})
	return
}

// VerifyOTP verify otp
func VerifyOTP(c *gin.Context) {
	phoneNumber := c.Query("phone_number")
	otp := c.Query("otp")
	err := validatePhoneNumber(phoneNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error while verifying OTP.",
		})
		return
	}
	val, err := database.RedisClient.Get(phoneNumber).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error while verifying OTP.",
		})
	} else {
		data := &PhoneVerificationData{}
		err := json.Unmarshal([]byte(val), data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error while verifying OTP.",
			})
			return
		}
		if data.Otp == otp {
			newData := PhoneVerificationData{
				phoneNumber,
				true,
				otp,
				time.Now(),
				data.SentTimestamp,
			}
			dataStr, err := json.Marshal(newData)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Error while verifying OTP.",
				})
				return
			}
			_, err = database.RedisClient.Set(phoneNumber, string(dataStr), 0).Result()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Error while verifying OTP.",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "verified",
			})
			fmt.Println("Otp verified for phoneNumber " + phoneNumber + " is : " + otp)
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Wrong OTP",
			})
		}
	}
}

func validatePhoneNumber(phoneNumber string) error {
	numOfDigits := len(phoneNumber)
	errorString := "phone number wrong"
	if numOfDigits != 10 {
		return errors.New(errorString)
	}
	re := regexp.MustCompile(`^[1-9]\d{9}$`)
	if !re.MatchString(phoneNumber) {
		return errors.New(errorString)
	}
	return nil
}

func rangeIn(low, hi int) string {
	return strconv.Itoa(low + rand.Intn(hi-low))
}
