package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"strings"
	"time"
	"math/rand"
	"encoding/json"
)

var (
	apiKey = func () string {
		return os.Getenv("API_KEY")
	}	
	ids map[string]func() func(http.ResponseWriter) bool = make(map[string]func() func(http.ResponseWriter) bool)
	count map[string]int = make(map[string]int)
)
const maxResults = 10

type Result struct {
	Id string `json:"id"`
	Indices []int `json:"indices"`
	Page string `json:"page"`
	Site string `json:"site"`
	Rank int `json:"rank"`
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/id/{id}", getId)
	r.HandleFunc("/search", search).Queries("primary", "", "secondary", "")
	http.ListenAndServe(":8080", handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r))
}

// /id/{id} endpoint
// Checks if the given id is in the map and runs the
// function the map points to
func getId (w http.ResponseWriter, r *http.Request) {
	// get id from url
	vars := mux.Vars(r)
	id := vars["id"]
	// get function from map
	getResult, ok := ids[id]
	if !ok {
		w.Write([]byte("Incorrect ID"))
		return
	}
	// run function
	moreResults := getResult()(w)
	if !moreResults {
		delete(ids, id)
	}
}

// /search endpoint
// Runs the search and returns the id of the results
// (the id which will be used to retrieve the results)
func search (w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	id := randomString(16)
	runSearches(vars.Get("primary"), vars.Get("secondary"), w, id)
	rs := Result{Id: id}
	mp := make(map[string]Result)
	mp["data"] = rs
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(mp); err != nil {
		panic(err)
	}
}

//
func runSearches(primary, secondary string, w http.ResponseWriter, id string) {
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

	results := make(chan Result)
	createCallback(results, id)

	for i, link := range resp.Items {
		go runSearch(link, results, secondary, i)
	}
}

func createCallback(results chan Result, id string) {
	var resultsReturned int
	ids[id] = func() func(http.ResponseWriter) bool {
		return func(w http.ResponseWriter) bool {
			fmt.Println(resultsReturned)
			if resultsReturned >= maxResults {
				w.Write([]byte("No More!"))
				return false
			}
			resultsReturned++
			var res Result
			res = <- results
			js, err := json.Marshal(res)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return false
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
			return true
		}
	}
}

func runSearch(link *customsearch.Result, resultsChan chan Result,
	secondary string, rank int) {

	body := getLinkBody(link.Link)
	
	fmt.Printf("\nWEBSITE: %s  Length: %d\n", link.Link, len(body))
	
	indices := getReferences(body, secondary)
	result := Result{
		Indices: indices,
		Page: body,
		Site: link.Link,
		Rank: rank,
	}
	// ids[id] = resultsChan
	// ids[id] <- result
	resultsChan <- result

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
func getReferences(page, key string) []int {
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
	//references := getQuotes(page, indices, len(key), 60)
	return indices
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

func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
