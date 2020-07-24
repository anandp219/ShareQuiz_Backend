package app

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sharequiz/app/database"
	"sharequiz/app/thirdparty"
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
	err := ValidatePhoneNumber(phoneNumber)
	success := false
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
		fmt.Println("Otp sent for phoneNumber " + phoneNumber + " is : " + otp)
		success = thirdparty.SendSms(phoneNumber, otp)
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
		if data.IsVerified == true || data.SentTimestamp.After(time.Now().Add(time.Duration(-1)*time.Second)) {
			errorMessage := "Phone number already in use."
			if !data.IsVerified {
				errorMessage = "Please wait sometime before requesting otp again"
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": errorMessage,
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
		success = thirdparty.SendSms(phoneNumber, otp)
	}
	if success {
		c.JSON(http.StatusOK, gin.H{
			"result": "success",
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error while sending OTP.",
		})
		return
	}
	return
}

// VerifyOTP verify otp
func VerifyOTP(c *gin.Context) {
	phoneNumber := c.Query("phone_number")
	otp := c.Query("otp")
	err := ValidatePhoneNumber(phoneNumber)
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

//ValidatePhoneNumber validate the phone number
func ValidatePhoneNumber(phoneNumber string) error {
	// numOfDigits := len(phoneNumber)
	// errorString := "phone number wrong"
	// if numOfDigits != 10 {
	// 	return errors.New(errorString)
	// }
	// re := regexp.MustCompile(`^[1-9]\d{9}$`)
	// if !re.MatchString(phoneNumber) {
	// 	return errors.New(errorString)
	// }
	return nil
}

func rangeIn(low, hi int) string {
	return strconv.Itoa(low + rand.Intn(hi-low))
}
