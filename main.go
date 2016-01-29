package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"github.com/gorilla/mux"
	"fmt"
	"log"
	"net/http"
//	"net/url"
	"os"
	"io/ioutil"
	"strings"
//	"encoding/json"
)

var apiKey = func () string {
	return os.Getenv("API_KEY")
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/id/{id}", getId)
	r.HandleFunc("/search", search).Queries("primary", "", "secondary", "")

	http.ListenAndServe(":8080", r)
	
	// client := &http.Client{
	// 	Transport: &transport.APIKey{Key: apiKey()},
	// }
	// customSearchService, err := customsearch.New(client)
	// if err != nil {
	// 	log.Fatal(err)
	// 	return
	// }
	// listCall := customSearchService.Cse.List("star wars")
	// listCall.Cx("001559197599027589089:09osstjowqu")
	
	// search, err :=listCall.Do()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// print links
	// for _, link := range search.Items {
		
	// 	body := getLinkBody(link.Link)
	// 	fmt.Printf("\nWEBSITE: %s  Length: %d\n", link.Link, len(body))
		
	// 	refs := getReferences(body, "Luke")
	// 	for _, i := range refs {
	// 		fmt.Println("<<<   " + i + "   >>>")
	// 	}
	// }
	// go get links and stuff and search them
}

func getId (w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GetIDHERE"))
}

func search (w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	createCall(vars.Get("primary"), vars.Get("secondary"))
	w.Write([]byte("Search: " + vars.Get("primary")))
}

func createCall(primary, secondary string) {
	client := &http.Client{
		Transport: &transport.APIKey{Key: apiKey()},
	}
	customSearchService, err := customsearch.New(client)
	if err != nil {
		log.Fatal(err)
		return
	}
	listCall := customSearchService.Cse.List(primary)
	listCall.Cx("001559197599027589089:09osstjowqu")
	
	resp, err :=listCall.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, link := range resp.Items {
		
		body := getLinkBody(link.Link)
		fmt.Printf("\nWEBSITE: %s  Length: %d\n", link.Link, len(body))
		
		refs := getReferences(body, secondary)
		for _, i := range refs {
			fmt.Println("<<<   " + i + "   >>>")
		}
	}
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

// find single reference to key and get surounding context and rest of body to resp
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
