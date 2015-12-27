package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"io/ioutil"

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
		
		body := getLinkBody(link.Link)
		fmt.Printf("\nWEBSITE: %s  Length: %d\n", link.Link, len(body))
		
		refs := getReferences(body, "Luke")
		for _, i := range refs {
			fmt.Println("<<<   " + i + "   >>>")
		}
	}

	// go get links and stuff and search them
	
}

// fetch the actual page. maybe only get the body of html? maybe not
func getLinkBody(link string) string {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)

	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	// TODO check to see if there is better way to convert byte slice to string
	return string(body[:])
}

// get all the references to the key string
func getReferences(page, key string) []string {
	count := strings.Count(page, key)
//	references := make([]string, count)
	rest := page
	index := 0
	indices := make([]int, 1)
	for i := 0; i < count; i++ {
		rest, index = getReferenceIndex(rest, key, index)
		indices = append(indices, index)
		//references = append(references, reference)
	}
	references := getQuotes(page, indices, len(key), 60)
	return references
}

// find single reference to key and get surounding context and rest of body to search
func getReferenceIndex(rest, key string, pageLenDif int) (string, int) {
	index := strings.Index(rest, key)
	return rest[index+len(key):], pageLenDif + index + len(key)
}

func getQuotes(page string, indices []int, keyLen, contextLength int) []string {
	zero, leng := 0, len(page)
	quotes := make([]string, 0)
	for i := 0; i < len(indices); i++ {
		back, front := indices[i]-contextLength, indices[i]+contextLength+keyLen
		if indices[i] == -1 || indices[i] == 0 {
			continue
		}
		if front > leng {
			front = leng
		}
		if back < zero {
			back = zero
		}
		quotes = append(quotes, page[back:front])
	}
	return quotes
}
