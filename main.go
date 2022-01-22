package main

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/uptrace/bunrouter"
	"github.com/uptrace/bunrouter/extra/reqlog"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const ApiTimeout = 1500 // 毫秒

var client = http.Client{
	Timeout: ApiTimeout * time.Millisecond,
}

func main() {
	// https://studygolang.com/articles/28263
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		log.Fatal("defaultRoundTripper not an *http.Transport")
	}
	defaultTransport := *defaultTransportPointer
	defaultTransport.MaxConnsPerHost = 100
	defaultTransport.MaxIdleConnsPerHost = 100 / 2

	router := bunrouter.New(
		bunrouter.WithMiddleware(reqlog.NewMiddleware(
			reqlog.FromEnv("BUNDEBUG"),
		)),
	)

	router.GET("/", func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Println("hello")
		return nil
	})

	// 模拟外部也许不能及时响应的API
	router.GET("/heavy", func(w http.ResponseWriter, req bunrouter.Request) error {
		val := rand.Intn(5)
		time.Sleep(time.Second * time.Duration(val))
		_, err := w.Write([]byte(strconv.FormatInt(int64(val), 10)))
		return err
	})

	router.GET("/call", func(w http.ResponseWriter, req bunrouter.Request) error {
		apiUrl := "http://localhost:9999/heavy" // 外部API
		// TODO 增加body
		api, err := http.NewRequestWithContext(req.Context(), "GET", apiUrl, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(api)
		if err != nil {
			msg := err.Error()
			if strings.Index(msg, "Client.Timeout") > 0 {
				w.Write([]byte("Timeout ..."))
			}
			return err
		}
		defer resp.Body.Close()
		buffer, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_, err = w.Write(buffer)
		return err
	})

	err := http.ListenAndServe(":9999", router)
	if err != nil {
		log.Error("Can't Open Web")
	}

}
