package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

//elasticClient elastic client used for elastic search access
var elasticClient *elasticsearch.Client
var indexName = "questions"

//InitElastic function initialises the
func InitElastic() {
	var err error
	cfg := elasticsearch.Config{
		Addresses: []string{
			os.Getenv("ELASTIC_URL"),
		},
	}
	elasticClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		log.Panicln(err)
	}
}

//SearchQuestions is used to search for questions
func SearchQuestions(searchQuery map[string]interface{}) (map[string]interface{}, error) {
	if elasticClient == nil {
		return nil, errors.New("empty elastic client")
	}

	var buf bytes.Buffer
	var result map[string]interface{}

	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, err
	}

	res, err := elasticClient.Search(
		elasticClient.Search.WithContext(context.Background()),
		elasticClient.Search.WithIndex(indexName),
		elasticClient.Search.WithBody(&buf),
		elasticClient.Search.WithTrackTotalHits(true),
		elasticClient.Search.WithPretty(),
	)

	if err != nil || res.IsError() {
		return nil, err
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	log.Println(result["hits"].(map[string]interface{}))

	return result["hits"].(map[string]interface{}), nil
}
