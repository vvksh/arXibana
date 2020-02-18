package main

import (
	// "context"

	// "encoding/json"
	// "io/ioutil"
	// "log"
	// "net/http"
	// "strconv"
	// "strings"
	// "github.com/elastic/go-elasticsearch"
	// "github.com/elastic/go-elasticsearch/esapi"

	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/mmcdole/gofeed"

	// "github.com/mmcdole/gofeed/rss"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/google/uuid"
)

const (
	indexName string = "test8"
	mapping   string = `{
		"properties" : {
				"title": {
					"type": "text",
					"fielddata": true
				},
				"description" : {
					"type": "text",
					"fielddata": true
				},
				"link" : {
					"type": "keyword"
				},
				"author" : {
					"type": "keyword"
				},
				"categories" : {
					"type": "text"
				}
		}
	}`
)

type ArxivItem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Link        string   `json:"link"`
	Author      string   `json:"author"`
	Categories  []string `json:"categories"`
}

func main() {
	log.Println("started arxivprocessing")
	url := "http://export.arxiv.org/api/query?search_query=cat:cs.DB&start=0&max_results=10"
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
		// ...
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error while initiazing elastic client: %s", err)
	}

	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(url)
	// fmt.Println(feed.String())

	createIndexIfNotPresent(es)

	for _, item := range feed.Items {
		newArxivItem := ArxivItem{
			item.Title,
			item.Description,
			item.Link,
			item.Author.Name,
			item.Categories}
		jsonItem, _ := json.Marshal(&newArxivItem)

		// fmt.Println(string(jsonItem))
		publishToElastic(string(jsonItem), es)

	}
}

func createIndexIfNotPresent(ElasticClient *elasticsearch.Client) {
	log.Println("creating index")

	existsRequest := esapi.IndicesExistsRequest{
		Index: []string{indexName}}

	existsResponse, err := existsRequest.Do(context.Background(), ElasticClient)
	if err != nil {
		panic(err)
	}
	if existsResponse.StatusCode == 200 {
		fmt.Println("index already exists")
		return
	}
	log.Println("Index not found, creating one")

	indexCreateRequest := esapi.IndicesCreateRequest{
		Index: indexName}

	createResponse, err := indexCreateRequest.Do(context.Background(), ElasticClient)

	if err != nil {
		fmt.Println("cannot create index")
		fmt.Println(err)
		return
	}

	log.Println(createResponse)

	putMappingRequest := esapi.IndicesPutMappingRequest{
		Index:        []string{indexName},
		DocumentType: "_doc",
		Body:         strings.NewReader(mapping)}

	res, err := putMappingRequest.Do(context.Background(), ElasticClient)

	if err != nil {
		log.Panicln(err)
	}
	log.Println("Mapping successful", res)
}

func publishToElastic(Jsonitem string, ElasticClient *elasticsearch.Client) {
	log.Println("Publishing" + Jsonitem)
	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: uuid.New().String(),
		Body:       strings.NewReader(Jsonitem),
		Refresh:    "true",
	}

	// // Perform the request with the client.
	res, err := req.Do(context.Background(), ElasticClient)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("[%s] Error indexing document ID=%d", res.Status(), 1)
	} else {
		// 	// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			log.Printf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and indexed document version.
			log.Printf("[%s] %s; version=%d", res.Status(), r["result"], int(r["_version"].(float64)))
		}
	}
}
