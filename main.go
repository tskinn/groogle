package main

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"strings"
	"time"
	"math/rand"
	"encoding/json"
	"runtime"
	"net/url"
	"regexp"
	"golang.org/x/net/html/atom"
	. "html"
	"./strip"
)

var (
	ids = make(map[string]func() func(http.ResponseWriter) bool)
	count = make(map[string]int)
)
const maxResults = 10

type Result struct {
	Id string        `json:"id"`
	Indices []int    `json:"indices"`
	Results []string `json:"results"`
	Page string      `json:"page"`
	Site string      `json:"site"`
	Rank int         `json:"rank"`
	Search string    `json:"search"`
}

func main() {

	r := mux.NewRouter()
	// r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	r.HandleFunc("/id/{id}", getId)
	r.HandleFunc("/search", search).Queries("primary", "", "secondary", "")
	http.ListenAndServe(":8080", handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r))
}

func writeResult(w http.ResponseWriter, rs Result) {

	mp := make(map[string]Result)
	mp["data"] = rs
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(mp); err != nil {
		fmt.Println("error: ", err)// TODO
	}

}

// /id/{id} endpoint
// Checks if the given id is in the map and runs the
// function the map points to
func getId (w http.ResponseWriter, r *http.Request) {
	fmt.Println("NumGoroutines: ", runtime.NumGoroutine())
	// get id from url
	vars := mux.Vars(r)
	id := vars["id"]
	// get function from map
	getResult, ok := ids[id]
	if !ok {
		writeResult(w, Result{})
		return
	}
	// run function
	moreResults := getResult()(w)
	if !moreResults {
//		writeResult(w, Result{})
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
	writeResult(w, Result{Id: id})
	runSearches(vars.Get("primary"), vars.Get("secondary"), id)
	
}

//
func runSearches(primary, secondary string, id string) {
	results := make(chan Result)
	links := getLinks(primary)
	createCallback(results, id, len(links))
	
	for i, link := range links {
		go runSearch(link, results, secondary, i)
	}
}

func createCallback(results chan Result, id string, numResults int) {
	var resultsReturned int
	var totalResults = numResults
	ids[id] = func() func(http.ResponseWriter) bool {
		return func(w http.ResponseWriter) bool {
			//fmt.Println(resultsReturned)
			if resultsReturned == totalResults {
				writeResult(w, Result{})
				close(results)
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
	// res := getContainingNodes(secondary, body)
	body = removeDoctype(UnescapeString(strip.StripTags(body)))
	indices := getReferences(body, secondary)
	result := Result{
		Indices: indices,
		Page: body,
		Site: link,
		Rank: rank,
		Search: secondary,
	}

	resultsChan <- result
}

func removeDoctype(s string) string {
	// if s[0] != '<' {
	// 	return s
	// }
	end := strings.IndexByte(s, '>')
	if end > 0 {
		return s[end + 1:]
	}
	return s
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
	return string(body)
}

// get all the references to the key string
func getReferences(page, key string) []int {
	reg := regexp.MustCompile(regexify(key))
	indi := reg.FindAllIndex([]byte(page), -1)
	indices := make([]int, 0)
	for _, i := range indi {
		indices = append(indices, i[0])
	}
	return indices
}

func getContainingNodes(s, body string) []string {

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	//nodes := make([]html.Node, 0)
	nodes := make([]string, 0)
	reg := regexp.MustCompile(regexify(s))
	var f func(*html.Node)
	f = func(n *html.Node) {
		if reg.MatchString(n.Data) {
			str := ""
			p := n.Parent
			// str += getData(p)
			str += strip.StripTags(p.Data)
			nodes = append(nodes, str)
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nodes
}

func getData(n *html.Node) string {
	data := ""
	if n == nil {
		return ""
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		data += c.Data + getData(c)
	}
	return data
}

func isAtom(a atom.Atom) bool {
	if a == atom.A    ||
		a == atom.I    ||
		a == atom.P    ||
		a == atom.H1   ||
		a == atom.H2   ||
		a == atom.H3   ||
		a == atom.H4   ||
		a == atom.H5   ||
		a == atom.H6   ||
		a == atom.Div {
		return true
	}
	return false
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

// create a randome string for an id
// TODO could be better...
func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// GET search results are parse the results page
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

// The links to the actuall results are a little weird so clean them up
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
		}
	}
	return ns
}

func regexify(s string) string {
	reg := ""
	for _, i := range s {
		tmp := "[" + strings.ToLower(fmt.Sprintf("%c", i)) + "|" + strings.ToUpper(fmt.Sprintf("%c", i)) + "]"
		reg += tmp
	}
	return reg
}
