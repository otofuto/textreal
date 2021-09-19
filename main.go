package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var port string

var clients = make(map[*websocket.Conn]string)
var broadcast = make(chan SocketMessage)
var upgrader = websocket.Upgrader{}

type SocketMessage struct {
	Message string `json:"message"`
	RoomId  string `json:"room_id"`
}

func main() {
	godotenv.Load()
	port = os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	http.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", IndexHandle)
	http.HandleFunc("/upload/", UploadHandle)

	http.HandleFunc("/ws/", SocketHandle)
	go handleMessages()

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if r.Method == http.MethodGet {
		temp := template.Must(template.ParseFiles("template/index.html"))
		if err := temp.Execute(w, ""); err != nil {
			log.Println(err)
			http.Error(w, "HTTP 500 Internal server error", 500)
			return
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func UploadHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		r.ParseMultipartForm(32 << 20)
		savedFiles := make([]string, 0)
		fileHeaders := r.MultipartForm.File["file"]
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				log.Println("ファイル見つからない")
				log.Println(err)
				http.Error(w, "upload failed", 500)
				return
			}

			save, err := os.Create("./static/uploaded/" + fileHeader.Filename)
			if err != nil {
				fmt.Println("ファイル確保失敗")
				log.Println(err)
				http.Error(w, "upload failed", 500)
				return
			}

			defer save.Close()
			defer file.Close()
			_, err = io.Copy(save, file)
			if err != nil {
				log.Println("ファイル保存失敗")
				log.Println(err)
				http.Error(w, "upload failed", 500)
				return
			}
			savedFiles = append(savedFiles, fileHeader.Filename)
		}
		bytes, _ := json.Marshal(savedFiles)
		fmt.Fprintf(w, string(bytes))
	} else if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		files, err := ioutil.ReadDir("./static/uploaded")
		if err != nil {
			log.Println(err)
			http.Error(w, "ファイル一覧の取得に失敗しました。", 500)
			return
		}
		paths := make([]string, 0)
		for _, file := range files {
			if !file.IsDir() && file.Name() != ".gitkeep" {
				paths = append(paths, file.Name())
			}
		}
		bytes, _ := json.Marshal(paths)
		fmt.Fprintf(w, string(bytes))
	} else {
		w.Header().Set("Content-Type", "text/html")
		http.Error(w, "このURLではPOSTメソッド、GETメソッドのみに対応しています。", 405)
	}
}

func SocketHandle(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r2 *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	clients[ws] = r.URL.Path[len("/ws/"):]

	for {
		var msg SocketMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		msg.RoomId = r.URL.Path[len("/ws/"):]
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client, id := range clients {
			if id == msg.RoomId {
				err := client.WriteJSON(msg)
				if err != nil {
					log.Printf("error: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
	}
}
