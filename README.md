Session implementation by Golang
--------------------------------
Usage:
```go
package main

import(
	_ "github.com/liexusong/go-session/redis"
	session "github.com/liexusong/go-session"
	"net/http"
	"log"
)

var sessionConfig = &session.Config{
    SavePath:       "tcp://127.0.0.1:6379",
    SessionName:    "session_id",
    CookieDomain:   "test.com",
    CookieLifetime: 0,
    GCProbability:  1,
    GCDivisor:      1,
    GCMaxLifetime:  100,
}

func SessionTestHandler(w http.ResponseWriter, r *http.Request) {
	se, err := session.NewSession(w, r, sessionConfig)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	err = se.Set("name", "liexusong")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	var name string

	err = se.Get("name", &name)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(name))
}

func main() {
    http.HandleFunc("/", SessionTestHandler)

    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
```
