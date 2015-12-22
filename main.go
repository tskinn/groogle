package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"fmt"
	"log"
	"net/http"
	"os"
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
}
