package web

import "net/http"

func sendResponse(w http.ResponseWriter, response []byte, contentType string, statusCode int) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	w.Write(response)
}
