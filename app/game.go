package app

import (
	"encoding/json"
	"errors"
	"sharequiz/app/database"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

//Question object
type Question struct {
	QuestionText  string            `json:"questionText"`
	Options       []string          `json:"options"`
	Answer        string            `json:"answer"`
	PlayerAnswers map[string]string `json:"playerAnswers"`
}

// Game object status 1 is active, 2 is Disconnected and 3 is Finished
type Game struct {
	ID               string            `json:"id"`
	Language         Language          `json:"language"`
	MaxQuestions     int               `json:"maxQuestions"`
	NumberOfPlayers  int               `json:"numberOfPlayers"`
	QuestionNumber   int               `json:"questionNumber"`
	Players          map[string]Player `json:"players"`
	Status           Status            `json:"status"`
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Questions        []Question        `json:"questions"`
	Scores           map[string][]int  `json:"scores"`
}

// Player object
type Player struct {
	ID       string `json:"id"`
	Score    int    `json:"score"`
	Selected int    `json:"selected"`
}

// CreateGame function
func CreateGame(maxQuestions int, language Language, numberOfPlayers int, topic Topic) (string, error) {
	time.Now()
	// 3 tries to create game
	for i := 1; i <= 3; i++ {
		gameID := 0
		lastGameID, err := database.RedisClient.Get(LastGameIDKey).Result()
		if err == redis.Nil {
			gameID = 1
		} else if err != nil {
			continue
		} else {
			gameID, _ = strconv.Atoi(lastGameID)
			gameID++
		}

		questions, err := GetGameQuestions(topic, language, maxQuestions)
		if err != nil {
			continue
		}

		data := Game{
			ID:               strconv.Itoa(gameID),
			Language:         language,
			MaxQuestions:     maxQuestions,
			NumberOfPlayers:  numberOfPlayers,
			QuestionNumber:   0,
			Players:          make(map[string]Player),
			Status:           Active,
			CreatedTimestamp: time.Now().Unix(),
			Questions:        questions,
			Scores:           nil,
		}
		dataStr, err := json.Marshal(data)
		if err != nil {
			continue
		}

		_, err = database.RedisClient.Set(strconv.Itoa(gameID), string(dataStr), 0).Result()
		if err == nil {
			_, err := database.RedisClient.Set(LastGameIDKey, gameID, 0).Result()
			if err == nil {
				return string(gameID), nil
			}
		}
	}
	return "error", errors.New("Error while creating game for the user")
}

//GetGameQuestions get game questions
func GetGameQuestions(topic Topic, language Language, numOfQuestions int) ([]Question, error) {
	randomScoreQuery := map[string]interface{}{
		"random_score": map[string]interface{}{},
	}
	functionsMap := []map[string]interface{}{randomScoreQuery}

	topicsArray := []string{strings.ToLower(topic.String())}
	topicsQuery := map[string]interface{}{
		"terms": map[string][]string{
			"topics": topicsArray,
		},
	}

	languageQuery := map[string]interface{}{
		"term": map[string]string{
			"language": strings.ToLower(language.String()),
		},
	}

	filterQuery := []map[string]interface{}{topicsQuery, languageQuery}

	query := map[string]interface{}{
		"size": numOfQuestions,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filterQuery,
				"must": map[string]interface{}{
					"function_score": map[string]interface{}{
						"functions": functionsMap,
					},
				},
			},
		},
	}

	result, err := database.SearchQuestions(query)
	questions := make([]Question, numOfQuestions)
	hits := result["hits"].([]interface{})
	for i, hit := range hits {
		questionObject := hit.(map[string]interface{})["_source"].(map[string]interface{})
		question := Question{
			QuestionText:  questionObject["question_text"].(string),
			Answer:        questionObject["answer"].(string),
			Options:       getOptionsArray(questionObject["options"].([]interface{})),
			PlayerAnswers: make(map[string]string),
		}
		questions[i] = question
	}

	if err != nil {
		return nil, err
	}

	return questions, nil
}

func getOptionsArray(optionsInterface []interface{}) []string {
	optionString := make([]string, len(optionsInterface))
	for i, option := range optionsInterface {
		optionString[i] = option.(string)
	}
	return optionString
}
