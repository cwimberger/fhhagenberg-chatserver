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

	return !strings.ContainsAny(text, "\n\r\t")
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

	send(w, &Message{Text: "Welcome to hagenberg chat!", Type: "welcome"})
	broadcast(&Message{Text: email + " joined the chat.", Type: "join"})

	defer broadcast(&Message{Text: email + " left the chat.", Type: "leave"})

	msgChan := make(chan *Message)
	c := &ChatClient{MsgChan: msgChan}
	clients = append(clients, c)

	for {
		msg := <-msgChan
		send(w, msg)
	}
}

func send(w http.ResponseWriter, message *Message) {
	b, err := json.Marshal(message)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	w.Write(b)
	w.Write([]byte("\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
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

	broadcast(&Message{Email: email, Text: text, Type: typ})
}

func broadcast(message *Message) {
	for _, c := range clients {
		c.MsgChan <- message
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
