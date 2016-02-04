package main

import (
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/googleapi/transport"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"fmt"
	"log"
	"net/http"
//	"net/url"
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
	
	ids map[string]chan Result = make(map[string]chan Result)
)

type Result struct {
	Indices []int
	Page string
	Site string
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/id/{id}", getId)
	r.HandleFunc("/search", search).Queries("primary", "", "secondary", "")
	http.ListenAndServe(":8080", handlers.CORS(handlers.AllowedOrigins([]string{"localhost"}))(r))
}

func getId (w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ch, ok := ids[id]
	if !ok {
		w.Write([]byte("Incorrect ID"))
		return
	}
	var res Result
	res =<- ch
	js, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func search (w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	id := randomString(16)
	w.Write([]byte(id))
	runSearches(vars.Get("primary"), vars.Get("secondary"), w, id)
	w.Write([]byte("\nSearch: " + vars.Get("primary")))
}

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
	
	for _, link := range resp.Items {
		go runSearch(link, results, secondary, id)
		// make all on function
	}
}

func runSearch(link *customsearch.Result, resultsChan chan Result,
	secondary, id string) {

	body := getLinkBody(link.Link)
	
	fmt.Printf("\nWEBSITE: %s  Length: %d\n", link.Link, len(body))
	
	indices := getReferences(body, secondary)
	result := Result{
		Indices: indices,
		Page: body,
		Site: link.Link,
	}
	ids[id] = resultsChan
	ids[id] <- result
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
