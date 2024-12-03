package thirdparty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var accountSid = ""
var authToken = ""
var urlStr = "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"
var twilioPhoneNumber = ""

// SendSms sends the sms for OTP verification
func SendSms(phoneNumber string, otp string) bool {
	if os.Getenv("ENV") == "local" {
		return true
	}
	msgData := url.Values{}
	msgData.Set("To", phoneNumber)
	msgData.Set("From", twilioPhoneNumber)
	msgData.Set("Body", "otp for Sharequiz is "+otp)
	msgDataReader := *strings.NewReader(msgData.Encode())
	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := client.Do(req)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&data)
		if err == nil {
			fmt.Println(data["sid"])
		}
		return true
	}
	fmt.Println(resp.Status)
	return false
}
