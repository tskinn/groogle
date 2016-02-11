package main

import (
//	"google.golang.org/api/customsearch/v1"
//	"google.golang.org/api/googleapi/transport"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"github.com/PuerkitoBio/goquery"
//	"golang.org/x/net/html"
//	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"strings"
	"time"
	"math/rand"
	"encoding/json"
	"runtime"
	"net/url"
)

var (
	apiKey = func () string {
		return os.Getenv("API_KEY")
	}	
	ids = make(map[string]func() func(http.ResponseWriter) bool)
	count = make(map[string]int)
)
const maxResults = 10

type Result struct {
	Id string     `json:"id"`
	Indices []int `json:"indices"`
	Page string   `json:"page"`
	Site string   `json:"site"`
	Rank int      `json:"rank"`
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
	fmt.Println("NumGoroutines: ", runtime.NumGoroutine())
	vars := r.URL.Query()
	id := randomString(16)
	rs := Result{Id: id}
	mp := make(map[string]Result)
	mp["data"] = rs
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(mp); err != nil {
		fmt.Println("error: ", err)// TODO
	}
	runSearches(vars.Get("primary"), vars.Get("secondary"), id)
	
}

//
func runSearches(primary, secondary string, id string) {
	results := make(chan Result)
	createCallback(results, id)

	links := getLinks(primary)
	
	for i, link := range links {
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
			res.Id = id
			mp := make(map[string]Result)
			mp["data"] = res
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(mp); err != nil {
				fmt.Println("error: ", err)// TODO
			}
			
			return true
		}
	}
}

func runSearch(link string, resultsChan chan Result,
	secondary string, rank int) {

	body := getLinkBody(link)
	
	//fmt.Printf("\nWEBSITE: %s  Length: %d\n", link.Link, len(body))
	
	indices := getReferences(body, secondary)
	result := Result{
		Indices: indices,
		Page: body,
		Site: link,
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

func getLinks(s string) []string {
	v := url.Values{}
	v.Add("q", s)
	doc, err := goquery.NewDocument("https://www.google.com/search?" + v.Encode())
	if err != nil {
		fmt.Println("Error: ", err)// TODO
	}
	links := make([]string, 0)
	doc.Find(".r").Each(func (i int, s *goquery.Selection) {
		l, exists := s.Find("a").Attr("href")
		if exists {
			links = append(links, l)
		}
	})
	return pruneLinks(links)
}

func pruneLinks(s []string) []string {
	ns := make([]string, 0)
	for i := range s {
		first := strings.IndexByte(s[i], '=') + 1
		last := strings.IndexByte(s[i], '&')
		plast := strings.IndexByte(s[i], '%')
		if last > plast && plast != -1 {
			last = plast
		}
		if strings.Contains(s[i][first:last], "http") {
			ns = append(ns, s[i][first:last])
			fmt.Println(s[i][first:last])
		}
	}
	return ns
}
