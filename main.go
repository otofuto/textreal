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

type Docs struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Pass      string `json:"pass"`
	Text      string `json:"text"`
	Token     string `json:"token"`
	UpdatedAt string `json:"updated_at"`
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
			filename := "doc_view"
			cookie, err := r.Cookie("textreal_token")
			if err == nil {
				if doc.Token == cookie.Value {
					filename = "doc"
				}
			}
			if doc.Pass != "" && filename == "doc_view" {
				http.Redirect(w, r, "/", 303)
				return
			}
			temp := template.Must(template.ParseFiles("template/" + filename + ".html"))
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
			docs := make([]Docs, 0)
			db := database.Connect()
			defer db.Close()

			sql := "select `id`, `title`, `pass`, `updated_at` from `docs` order by `updated_at` limit 50"
			rows, err := db.Query(sql)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to fetch docs list", 500)
				return
			}
			defer rows.Close()

			for rows.Next() {
				var doc Docs
				err = rows.Scan(&doc.Id, &doc.Title, &doc.Pass, &doc.UpdatedAt)
				if err == nil {
					docs = append(docs, doc)
				}
			}
			temp := template.Must(template.ParseFiles("template/index.html"))
			if err := temp.Execute(w, struct {
				Docs []Docs
			}{
				Docs: docs,
			}); err != nil {
				log.Println(err)
				http.Error(w, "HTTP 500 Internal server error", 500)
				return
			}
		}
	} else if r.Method == http.MethodPost {
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
			filename := "doc_view"
			cookie, err := r.Cookie("textreal_token")
			if err == nil {
				if doc.Token == cookie.Value {
					filename = "doc"
				}
			}
			if doc.Pass != r.FormValue("pass") && filename == "doc_view" {
				http.Error(w, "パスコードが間違っています。", 405)
				return
			}
			temp := template.Must(template.ParseFiles("template/" + filename + ".html"))
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
			http.Error(w, "page not found", 404)
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func GetDoc(id int) (Docs, error) {
	db := database.Connect()
	defer db.Close()

	var ret Docs

	rows, err := db.Query("select `id`, `title`, `pass`, `text`, `token`, `updated_at` from `docs` where `id` = " + strconv.Itoa(id))
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&ret.Id, &ret.Title, &ret.Pass, &ret.Text, &ret.Token, &ret.UpdatedAt)
	} else {
		ret.Id = 0
	}
	return ret, nil
}

func UpdateDoc(id int, text, token string) error {
	db := database.Connect()
	defer db.Close()

	sql := "update `docs` set `text` = ? where `id` = ? and `token` = ?"
	upd, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	defer upd.Close()
	_, err = upd.Exec(&text, &id, &token)
	return err
}

func MakeHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		if title == "" {
			http.Error(w, "title is required", 400)
			return
		}

		var token string
		cookie, err := r.Cookie("textreal_token")
		if err != nil {
			token_bytes, err := bcrypt.GenerateFromPassword([]byte(time.Now().Format("yyyyMMddHHmmss")), 10)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to create hash", 500)
				return
			}
			token = string(token_bytes)
		} else {
			token = cookie.Value
		}

		db := database.Connect()
		defer db.Close()

		sql := "insert `docs` (`title`, `pass`, `text`, `token`) values (?, ?, '', ?)"
		ins, err := db.Prepare(sql)
		if err != nil {
			log.Println(err)
			http.Error(w, "sql insert error", 500)
			return
		}
		defer ins.Close()
		result, err := ins.Exec(&title, r.FormValue("pass"), &token)
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
		cookie = &http.Cookie{
			Name:     "textreal_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		fmt.Fprintf(w, string(bytes))
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func UpdateHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPut {
		cookie, err := r.Cookie("textreal_token")
		if err != nil {
			http.Error(w, "token is not set", 400)
			return
		}

		if r.FormValue("id") != "" {
			id, err := strconv.Atoi(r.FormValue("id"))
			if err != nil {
				http.Error(w, "id is not integer", 400)
				return
			}
			err = UpdateDoc(id, r.FormValue("text"), cookie.Value)
			if err != nil {
				log.Println(err)
				http.Error(w, "failed to update query", 500)
				return
			}
			fmt.Fprintf(w, "true")
			for client, wid := range clients {
				if wid == r.FormValue("id") {
					err := client.WriteJSON(SocketMessage{
						Message: r.FormValue("text"),
					})
					if err != nil {
						log.Printf("error: %v", err)
						client.Close()
						delete(clients, client)
					}
				}
			}
		} else {
			http.Error(w, "parameter 'id' is required", 400)
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
