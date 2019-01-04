Session implementation by Golang
--------------------------------
Usage:
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	session "github.com/liexusong/go-session"
	_ "github.com/liexusong/go-session/redis"
)

type User struct {
	Name string
	Age  int
	Sex  string
}

var sessionConfig = session.Config{
	SavePath:       "tcp://127.0.0.1:6379",
	SessionName:    "session_id",
	CookieDomain:   "localhost",
	CookieLifetime: 0,
	GCProbability:  1,
	GCDivisor:      1,
	GCMaxLifetime:  100,
}

var sessionManager *session.SessionManager

func setSessionHandler(w http.ResponseWriter, r *http.Request) {
	se := sessionManager.CreateSession(w, r)

	user := &User{
		Name: "liexusong",
		Age:  20,
		Sex:  "man",
	}

	err := se.Set("name", user)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte("OK"))
}

func getSessionHandler(w http.ResponseWriter, r *http.Request) {
	se := sessionManager.CreateSession(w, r)

	var user User

	err := se.Get("name", &user)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(fmt.Sprintf("%v", &user)))
}

func main() {
	var err error

	sessionManager, err = session.NewSessionManager(sessionConfig)
	if err != nil {
		log.Fatal("NewSessionManager:", err)
	}

	http.HandleFunc("/set", setSessionHandler)
	http.HandleFunc("/get", getSessionHandler)

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
```
