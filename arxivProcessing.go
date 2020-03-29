package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/mmcdole/gofeed"
)

const (
	defaultIndexName string = "test10"
	mapping          string = `{
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
				},
				"published" : {
					"type": "date"
				},
				"updated" : {
					"type": "date"
				}
		}
	}`

	MAX_SEED_RECORDS int        = 10000
	MAX_RESULTS_PER_SEED_CALL int = 2000
	MAX_RESULTS_PER_CALL int    = 10
	ELASTICSEARCH_URL string    = "http://elasticsearch:9200"
)

//ArxivItem  will get posted to elasticsearch
type ArxivItem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Link        string   `json:"link"`
	Author      string   `json:"author"`
	Categories  []string `json:"categories"`
	Published     string   `json:"published"`
	Updated     string   `json:"updated"`
}

func main() {
	log.Println("started arxivprocessing")

	// initialize feedparser, elastisearch client and create index if not present
	cfg := elasticsearch.Config{
	    Addresses: []string{
			ELASTICSEARCH_URL,
		},		
		RetryBackoff: simpleRetry,
		MaxRetries: 5,
		// ...
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error while initiazing elastic client: %s", err)
	}

	//If seed, pre-fetch articles and add to elasticsearch index, else only fetch recent
	seedPtr := flag.Bool("seed", false, "a bool")
	searchQueryPtr := flag.String("search_query", "cat:cs.DB+OR+cat:cs.DC", "search query for arxiv")
	indexNamePtr := flag.String("index_name", defaultIndexName, "indexName")
	flag.Parse()

	createIndexIfNotPresent(es, *indexNamePtr)
	fp := gofeed.NewParser()
	if *seedPtr {
		seed(fp, es, *searchQueryPtr, *indexNamePtr)
		return
	} else {
		fetchURLAndPublishToElastic(*indexNamePtr, getURL(0, MAX_RESULTS_PER_CALL, *searchQueryPtr), fp, es)
	}
}

func seed(feedParser *gofeed.Parser, elasticClient *elasticsearch.Client, searchQuery string, indexName string) {
	i := 0
	for i < MAX_SEED_RECORDS {
		url := getURL(i, MAX_RESULTS_PER_SEED_CALL, searchQuery)
		fetchURLAndPublishToElastic(indexName, url, feedParser, elasticClient)
		time.Sleep(1 * time.Second)
		i = i + MAX_RESULTS_PER_SEED_CALL
	}

}

func getURL(start int, max int, searchQuery string) string {
	return fmt.Sprintf("http://export.arxiv.org/api/query?search_query=%s&start=%d&max_results=%d&sortBy=lastUpdatedDate&sortOrder=descending", searchQuery, start, max)
}

func createIndexIfNotPresent(ElasticClient *elasticsearch.Client, indexName string) {
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

func fetchURLAndPublishToElastic(indexName string, url string, feedParser *gofeed.Parser, elasticClient *elasticsearch.Client) {
	feed, err := feedParser.ParseURL(url)
	if err != nil {
		log.Panicln("error while fetching arxiv url", err, url)
	}

	for _, item := range feed.Items {
		newArxivItem := ArxivItem{
			item.Title,
			item.Description,
			item.Link,
			item.Author.Name,
			item.Categories,
			item.Published,
			item.Updated,
		}
		jsonItem, _ := json.Marshal(&newArxivItem)
		docID := getDocID(item.Link)
		publishToElastic(indexName, docID, string(jsonItem), elasticClient)

	}
}

func getDocID(itemLink string) string {
	linkSplit := strings.Split(itemLink, "/")
	return linkSplit[len(linkSplit)-1]

}

func publishToElastic(indexName string, UUID string, Jsonitem string, ElasticClient *elasticsearch.Client) {
	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: UUID,
		Body:       strings.NewReader(Jsonitem),
		Refresh:    "true",
		OpType:     "create", //only add if absent
	}

	// // Perform the request with the client.
	res, err := req.Do(context.Background(), ElasticClient)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("[%s] Error indexing document ID=%s", res.Status(), UUID)
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

func simpleRetry(attempt int) time.Duration {
	return 60*time.Second;
}
