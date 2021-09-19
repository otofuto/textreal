package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/otofuto/textreal/pkg/database"
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
	http.HandleFunc("/make/", MakeHandle)
	http.HandleFunc("/update/", UpdateHandle)
	http.HandleFunc("/upload/", UploadHandle)

	http.HandleFunc("/ws/", SocketHandle)
	go handleMessages()

	log.Println("Listening on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if r.Method == http.MethodGet {
		id := r.URL.Path[len("/"):]
		if id != "" {
			idint, err := strconv.Atoi(id)
			if err != nil {
				http.Error(w, "id is not integer", 400)
				return
			}
			doc, err := GetDoc(idint)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to fetch doc", 500)
				return
			}
			temp := template.Must(template.ParseFiles("template/doc.html"))
			if err := temp.Execute(w, struct {
				Doc Docs
			}{
				Doc: doc,
			}); err != nil {
				log.Println(err)
				http.Error(w, "HTTP 500 Internal server error", 500)
				return
			}
		} else {
			temp := template.Must(template.ParseFiles("template/index.html"))
			if err := temp.Execute(w, ""); err != nil {
				log.Println(err)
				http.Error(w, "HTTP 500 Internal server error", 500)
				return
			}
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

type Docs struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Token string `json:"token"`
}

func GetDoc(id int) (Docs, error) {
	db := database.Connect()
	defer db.Close()

	var ret Docs

	rows, err := db.Query("select `id`, `title`, `text`, `token` from `docs` where `id` = " + strconv.Itoa(id))
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&ret.Id, &ret.Title, &ret.Text, &ret.Token)
	} else {
		ret.Id = 0
	}
	return ret, nil
}

func MakeHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		if title == "" {
			http.Error(w, "title is required", 400)
			return
		}

		token, err := bcrypt.GenerateFromPassword([]byte(time.Now().Format("yyyyMMddHHmmss")), 10)
		if err != nil {
			log.Println(err)
			http.Error(w, "failed to create hash", 500)
			return
		}

		db := database.Connect()
		defer db.Close()

		sql := "insert `docs` (`title`, `text`, `token`) values (?, '', ?)"
		ins, err := db.Prepare(sql)
		if err != nil {
			log.Println(err)
			http.Error(w, "sql insert error", 500)
			return
		}
		defer ins.Close()
		result, err := ins.Exec(&title, &token)
		newid64, err := result.LastInsertId()
		if err != nil {
			log.Println(err)
			http.Error(w, "failed to fetch newid", 500)
			return
		}
		res := struct {
			Id int64 `json:"id"`
		}{
			Id: newid64,
		}
		bytes, err := json.Marshal(res)
		if err != nil {
			http.Error(w, "failed to convert object to json", 500)
			return
		}
		fmt.Fprintf(w, string(bytes))
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func UpdateHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPut {
		//
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
