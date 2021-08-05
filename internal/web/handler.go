package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const MAX_URLS = 20

type Request struct {
	Urls []string `json:"urls"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("{%v} %v\n", r.Method, r.URL.Path)
	if r.Method == http.MethodPost && r.URL.Path == "/" {
		s.handleRequest(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404"))
}

type ErrorData struct {
	message    string
	statusCode int
}
type Result struct {
	Responses []string `json:"responses"`
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, time.Second)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	var body Request

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		sendResponse(w, []byte("wrong body format"), "text/plain", http.StatusBadRequest)
		return
	}
	if len(body.Urls) > MAX_URLS {
		sendResponse(
			w,
			[]byte(fmt.Sprintf("the maximum number of URLs allowed %v", MAX_URLS)),
			"text/plain",
			http.StatusBadRequest,
		)
		return
	}
	if len(body.Urls) == 0 {
		sendResponse(w, []byte("array of urls required"), "text/plain", http.StatusBadRequest)
		return
	}

	responses := make([]string, len(body.Urls))
	requests := make([]*http.Request, len(body.Urls))

	resultChan := make(chan []byte)
	errorChan := make(chan ErrorData)

	transport := http.Transport{
		Dial: dialTimeout,
	}

	client := http.Client{
		Transport: &transport,
	}

	go func() {
		for i, url := range body.Urls {
			wg.Add(1)
			go func(i int, url string) {
				defer wg.Done()

				var err error
				requests[i], err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					errorChan <- ErrorData{
						message:    "Internal server error",
						statusCode: http.StatusInternalServerError,
					}
					return
				}
				resp, err := client.Do(requests[i])
				if err != nil {
					if os.IsTimeout(err) {
						errorChan <- ErrorData{
							message:    fmt.Sprintf("request timeout on \"%v\"", url),
							statusCode: http.StatusRequestTimeout,
						}
					}
					errorChan <- ErrorData{
						message:    fmt.Sprintf("failed to get \"%v\": '%v'", url, err),
						statusCode: http.StatusBadGateway,
					}
					return
				}
				if resp.StatusCode >= 400 {
					errorChan <- ErrorData{
						message:    fmt.Sprintf("failed to get \"%v\" with status code: %v", url, resp.StatusCode),
						statusCode: resp.StatusCode,
					}
					return
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					errorChan <- ErrorData{
						message:    fmt.Sprintf("failed to process response from \"%v\"", url),
						statusCode: http.StatusInternalServerError,
					}
					return
				}
				responses[i] = string(body)
			}(i, url)
		}

		wg.Wait()

		result, err := json.Marshal(Result{responses})
		if err != nil {
			errorChan <- ErrorData{
				message:    "Internal server error",
				statusCode: http.StatusInternalServerError,
			}
		}
		resultChan <- result
		close(resultChan)
	}()

	select {
	case <-ctx.Done():
		sendResponse(w, []byte("request timeout"), "text/plain", http.StatusRequestTimeout)
	case msg := <-errorChan:
		sendResponse(w, []byte(msg.message), "text/plain", msg.statusCode)
	case result := <-resultChan:
		sendResponse(w, result, "application/json", http.StatusOK)
	}
}
