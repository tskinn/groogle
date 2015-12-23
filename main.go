package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
//	"encoding/json"
)

func main() {

	apiKey := os.Getenv("API_KEY")
	
	client := &http.Client{
		Transport: &transport.APIKey{Key: apiKey},
	}
	customSearchService, err := customsearch.New(client)
	if err != nil {
		log.Fatal(err)
		return
	}
	listCall := customSearchService.Cse.List("star wars")
	listCall.Cx("001559197599027589089:09osstjowqu")
	
	search, err :=listCall.Do()
	if err != nil {
		log.Fatal(err)
	}
	// print links
	for _, link := range search.Items {
		fmt.Println(link.Link)
	}

	// go get links and stuff and search them
	
}

// fetch the actual page. maybe only get the body of html? maybe not
func getLink() string {
	return ""
}

// get all the references to the key string
func getReferences(page, key string) []string {
	count := strings.Count(page, key)
	references := make([]string, count)
	rest := page
	reference := ""
	for i := 0; i < count; i++ {
		rest, reference = getReference(rest, key)
		references = append(references, reference)
	}
	return references
}

// find single reference to key and get surounding context and rest of body to search
func getReference(rest, key string) (string, string) {
	index := strings.Index(rest, key)
	return rest[index:], rest[index-10:index+10]
}
