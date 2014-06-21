package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ChatClient struct {
	Email   string
	MsgChan chan (*Message)
}

type Message struct {
	Email string `json:"email"`
	Text  string `json:"text"`
	Type  string `json:"type"`
}

var clients = []*ChatClient{}

func validateEmail(email string) bool {
	length := len(email)
	if length == 0 || length > 30 {
		return false
	}

	return !strings.ContainsAny(email, "\n\r\t ")
}

func validateText(text string) bool {
	length := len(text)
	if length == 0 || length > 255 {
		return false
	}

	return !strings.ContainsAny(text, "\n\r\t ")
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if !validateEmail(email) {
		http.Error(w, "email parameter invalid/missing", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	msgChan := make(chan *Message)
	c := &ChatClient{MsgChan: msgChan}
	clients = append(clients, c)

	for {
		msg := <-msgChan

		b, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("error:", err)
		}

		w.Write(b)
		w.Write([]byte("\n"))

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if !validateEmail(email) {
		http.Error(w, "email parameter invalid/missing", http.StatusBadRequest)
		return
	}

	text := r.FormValue("text")
	if !validateText(text) {
		http.Error(w, "text parameter invalid/missing", http.StatusBadRequest)
		return
	}

	typ := r.FormValue("type")
	if typ == "" {
		typ = "text"
	}
	if !validateText(typ) {
		http.Error(w, "type parameter invalid", http.StatusBadRequest)
		return
	}

	msg := &Message{
		Email: email,
		Text:  text,
		Type:  typ,
	}

	for _, c := range clients {
		c.MsgChan <- msg
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/post", postHandler)
	http.ListenAndServe(":"+port, nil)
}
