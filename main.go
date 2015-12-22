package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {

	apiKey := os.Getenv("API_KEY")
	
	client := &http.Client{
		Transport: &transport.APIKey{Key: apiKey},
	}
	fmt.Println("create service")
	customSearchService, err := customsearch.New(client)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println("create list")
	listCall := customSearchService.Cse.List("star wars")
	listCall.Cx("001559197599027589089:09osstjowqu")
	
	search, err :=listCall.Do()
	if err != nil {
		log.Fatal(err)
	}
	
	bites, err := search.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("done")
	//fmt.Print("%x", bites)
	s := string(bites[:len(bites)])
	fmt.Println(s)
}
