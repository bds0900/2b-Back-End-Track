package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type counters struct {
	sync.Mutex
	view  int
	click int
}
type counter struct {
	View  int `json:"view"`
	Click int `json:"click"`
}

type counterStore struct {
	sync.Mutex
	c map[string]map[string]counter
}
type configStore struct {
	sync.Mutex
	store map[string]config
}
type requestStore struct {
	sync.Mutex
	store map[string]map[time.Time]int
}

type config struct {
	windowMs   int
	max        int
	message    string
	statusCode int
}

type requestConter struct {
	requestTimeStamp time.Time
	requestCount     int
}

type response struct {
	Key   string  `json:"key"`
	Value counter `json:"value"`
}

var (
	c       = counters{}
	cc      = make(map[string]counter)                                //current counter
	prev    = make(map[string]counter)                                //previou counter
	cs      = configStore{store: make(map[string]config)}             //config store
	rs      = requestStore{store: make(map[string]map[time.Time]int)} //request store
	s       = counterStore{c: make(map[string]map[string]counter)}
	content = []string{"sports", "entertainment", "business", "education"}
	options = config{
		windowMs:   1 * 60 * 1000, // unit in millisecond
		max:        5,
		message:    "Too many request",
		statusCode: 429,
	}
)

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	data := content[rand.Intn(len(content))]

	c.Lock()
	c.view++
	c.Unlock()

	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(data)
	}

}

/*
Function : update
DESCRIPTION : This method copies current count store to previous count store
PARAMETERS : no parameter
RETURNS : no return
*/
func update() {
	//update prev at 59 second
	prev = cc
}

/*
Function : getIP
DESCRIPTION : Extract ip address from request
PARAMETERS : r *http.Request
RETURNS : string
*/
func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

/*
Function : getConfig
DESCRIPTION : Retrieve the configuration corresponding to the key. If no configuration exist, then use default configuration
PARAMETERS : key string
RETURNS : configuration config
*/
func getConfig(key string) config {
	configuration, exists := cs.store[key]
	if !exists {
		//default value
		cs.Lock()
		cs.store[key] = options
		cs.Unlock()
		return options
	}
	return configuration
}

/*
Function : getTotalWithinCurrentWindow
DESCRIPTION : Retrieve the request logs that corresponding to the key and return the total count within the window
PARAMETERS : key 		string
			 startTime time.Time
RETURNS : totalCount int
*/
func getTotalWithinCurrentWindow(key string, startTime time.Time) int {
	logs, exists := rs.store[key]
	if !exists {
		rs.Lock()
		rs.store[key] = make(map[time.Time]int)
		rs.Unlock()
		return 0
	}

	// if time stamp is after start time, add to total count
	// if time stamp is before start time, delete the log
	var totalCount int
	for timestamp, count := range logs {
		if timestamp.After(startTime) {
			totalCount += count
		} else {
			delete(logs, timestamp)
		}
	}
	return totalCount

}

/*
Function : registerRequest
DESCRIPTION : record the current time
PARAMETERS : key 		string
			 currentTime time.Time
RETURNS : no return
*/
func registerRequest(key string, currentTime time.Time) {
	rs.Lock()
	rs.store[key][currentTime]++
	rs.Unlock()
}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(data string) error {
	c.Lock()
	c.click++
	c.Unlock()
	cc[data] = counter{c.view, c.click}
	return nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	key := getIP(r)
	if !isAllowed(key) {
		w.WriteHeader(429)
		return
	}

	var slice []response
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	for _, val := range content {
		for k, v := range s.c[val] {
			key := fmt.Sprintf("%s:%s", val, k)
			p := response{key, v}
			slice = append(slice, p)
		}
	}
	json.NewEncoder(w).Encode(slice)

}

func isAllowed(key string) bool {
	currentTime := time.Now()
	config := getConfig(key)
	startTime := currentTime.Add(-time.Millisecond * time.Duration(config.windowMs))
	totalRequest := getTotalWithinCurrentWindow(key, startTime)
	if totalRequest >= config.max {
		return false
	}
	registerRequest(key, currentTime)
	return true
}

func uploadCounters() error {
	prevTime := time.Now().Add(-time.Minute * 1).Format("2006-01-02 03:04")
	currentTime := time.Now().Format("2006-01-02 03:04")

	s.Lock()
	for _, val := range content {
		if s.c[val] == nil {
			s.c[val] = make(map[string]counter)
		}
		s.c[val][prevTime] = counter{prev[val].View, prev[val].Click}
		s.c[val][currentTime] = counter{cc[val].View, cc[val].Click}
	}

	s.Unlock()

	return nil
}

func main() {
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/stats/", statsHandler)

	var left time.Duration = time.Duration(59 - time.Now().Second())
	ticker := time.NewTicker(left * time.Second)
	//update log every 5 second
	ticker2 := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				//fmt.Println("Tick at", t)
				ticker = time.NewTicker(time.Minute)
				go update()

			case <-ticker2.C:
				//fmt.Println("Tick at", t2)
				var err = uploadCounters()
				if err != nil {
					log.Fatal(err.Error())

				}
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
