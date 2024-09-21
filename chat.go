package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	_ "github.com/lib/pq"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Message struct {
	ID            int
	Text          string
	Time          string
	SenderLogin   string
	AccepterLogin string
}

func sendMessage(sender, receiver, message string) error {

	if sender == "" || receiver == "" || message == "" {
		return fmt.Errorf("sender, receiver, and message cannot be empty")
	}
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04")
	query := `
    INSERT INTO public.chat (text, "time", "senderLogin", "accepterLogin")
    VALUES ($1, $2, $3,$4)
    `

	_, err := db.Exec(query, message, formattedTime, sender, receiver)
	if err != nil {
		return fmt.Errorf("error while sending message: %v", err)
	}

	return nil
}

func getMessages(senderLogin, receiverLogin string) ([]Message, error) {
	if receiverLogin == "123" {
		fmt.Println("gg")
	}
	var messages []Message
	query := `
        SELECT id, text, "time", "senderLogin", "accepterLogin" 
        FROM public.chat 
        WHERE ("senderLogin" = $1 AND "accepterLogin" = $2) 
           OR ("senderLogin" = $2 AND "accepterLogin" = $1);`
	//log.Printf("senderLogin: %s, receiverLogin: %s", senderLogin, receiverLogin)

	rows, err := db.Query(query, senderLogin, receiverLogin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.Text, &msg.Time, &msg.SenderLogin, &msg.AccepterLogin)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

type Post struct {
	ID          int    `json:"id"`
	Time        string `json:"time"`
	Text        string `json:"text"`
	SenderLogin string `json:"sender_login"`
	Photo       []byte `json:"photo"`
}

func sendPost(senderLogin, text string, photo []byte) error {
	// If a photo was uploaded, store the photo path in the database
	if len(photo) > 0 {
		rand.Seed(time.Now().UnixNano())
		photoPath := filepath.Join("uploads", fmt.Sprintf("%d.jpg", rand.Int()))
		err := os.WriteFile(photoPath, photo, 0644)
		if err != nil {
			return err
		}
		// Now store the photoPath in the database
		_, err = db.Exec("INSERT INTO posts (\"time\", text, \"senderLogin\", photo) VALUES (NOW(), $1, $2, $3)", text, senderLogin, photoPath)
		if err != nil {
			return err
		}
	} else {
		_, err := db.Exec("INSERT INTO posts (\"time\", text, \"senderLogin\", photo) VALUES (NOW(), $1, $2, $3)", text, senderLogin, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func getPosts(db *sql.DB) ([]Post, error) {
	rows, err := db.Query("SELECT id, \"time\", text, \"senderLogin\", photo FROM posts ORDER BY id DESC LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		// Scan the photo column as a string, which will be the photo path
		err = rows.Scan(&p.ID, &p.Time, &p.Text, &p.SenderLogin, &p.Photo)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func chat(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {

		if r.FormValue("mess") != "" {
			senderLogin := r.FormValue("login")
			message := r.FormValue("mess")
			receiverLogin := r.FormValue("receiver")

			existsSender, err := userExists(senderLogin)
			if err != nil || !existsSender {
				http.Error(w, "Отправитель не зарегистрирован", http.StatusBadRequest)
				return
			}
			existsReceiver, err := userExists(receiverLogin)
			if err != nil || !existsReceiver {
				http.Error(w, "Получатель не зарегистрирован", http.StatusBadRequest)
				return
			}

			err = sendMessage(senderLogin, receiverLogin, message)
			if err != nil {
				http.Error(w, "Ошибка при отправке сообщения: "+err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, fmt.Sprintf("/chat?login=%s&receiver=%s", senderLogin, receiverLogin), http.StatusSeeOther)
			return
		}

		if r.FormValue("postText") != "" {
			postSender := r.FormValue("postSender")
			postText := r.FormValue("postText")
			photoFile, _, err := r.FormFile("photo")
			var photoData []byte

			if err == nil {
				defer photoFile.Close()

				photoData, err = io.ReadAll(photoFile)
				if err != nil {
					http.Error(w, "Ошибка при чтении фото: "+err.Error(), http.StatusInternalServerError)
					return
				}

				rand.Seed(time.Now().UnixNano())
				photoPath := filepath.Join("uploads", fmt.Sprintf("%d.jpg", rand.Int()))
				err = os.WriteFile(photoPath, photoData, 0644)
				if err != nil {
					http.Error(w, "Ошибка при сохранении фото: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}

			existsPostSender, err := userExists(postSender)
			if err != nil || !existsPostSender {
				http.Error(w, "Отправитель не зарегистрирован", http.StatusBadRequest)
				return
			}

			err = sendPost(postSender, postText, photoData)
			if err != nil {
				http.Error(w, "Ошибка при отправке поста: "+err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/chat", http.StatusSeeOther)
			return
		}
	}

	senderLogin := r.FormValue("login")
	receiverLogin := r.FormValue("receiver")

	messages, err := getMessages(senderLogin, receiverLogin)
	if err != nil {
		http.Error(w, "Ошибка при получении сообщений", http.StatusInternalServerError)
		return
	}

	posts, err := getPosts(db)
	if err != nil {
		http.Error(w, "Ошибка при получении постов", http.StatusInternalServerError)
		return
	}

	funcMap := template.FuncMap{
		"base64": func(b []byte) string {
			return base64.StdEncoding.EncodeToString(b)
		},
	}

	tmpl := `
    <!DOCTYPE html>
    <html>
    <head>
     <title>Чат</title>
    </head>
    <body>
     <h1>Чат</h1>
     <form method="POST" action="/chat">
        <label for="login">Логин отправителя:</label>
        <input type="text" id="login" name="login" value="{{.SenderLogin}}" required><br><br>
        <label for="receiver">Логин получателя:</label>
        <input type="text" id="receiver" name="receiver" value="{{.ReceiverLogin}}" required><br><br>
        <label for="mess">Сообщение:</label>
        <input type="text" id="mess" name="mess" required><br><br>
        <input type="submit" value="Отправить">
     </form>

      <!-- Post form -->
     <form method="POST" action="/chat" enctype="multipart/form-data">
        <label for="postSender">Логин отправителя:</label>
        <input type="text" id="postSender" name="postSender" required><br><br>
        <label for="postText">Текст поста:</label>
        <textarea id="postText" name="postText"></textarea><br><br>
        <input type="file" name="photo" accept="image/*"><br><br>
        <input type="submit" value="Отправить пост">
     </form>

     <h2>История сообщений:</h2>
     <ul>
        {{range .Messages}}
        <li><strong>ID:</strong> {{.ID}}, <strong>Text:</strong> {{.Text}}, <strong>Time:</strong> {{.Time}}, <strong>Sender:</strong> {{.SenderLogin}}, <strong>Accepter:</strong> {{.AccepterLogin}}</li>
        {{end}}
     </ul>

     <h2>Посты:</h2>
     <ul>
        {{range .Posts}}
        <li>
            <strong>ID:</strong> {{.ID}}, 
            <strong>Time:</strong> {{.Time}}, 
            <strong>Text:</strong> {{.Text}}, 
            <strong>Sender:</strong> {{.SenderLogin}}
            {{if .Photo}}
            <img src="/uploads/{{.Photo}}" alt="Photo" height="1000"> 
            {{end}}
        </li>
        {{end}}
     </ul>

    </body>
    </html>
    `

	t, err := template.New("chat").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}
	data := struct {
		Messages      []Message
		Posts         []Post
		SenderLogin   string
		ReceiverLogin string
	}{
		Messages:      messages,
		Posts:         posts,
		SenderLogin:   senderLogin,
		ReceiverLogin: receiverLogin,
	}
	t.Execute(w, data)
}

//func main() {
//	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
//		err = os.Mkdir("uploads", 0755)
//		if err != nil {
//			log.Fatal("Ошибка при создании папки uploads:", err)
//		}
//	}
//	initDB()
//	defer db.Close()
//	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
//	http.HandleFunc("/chat", chat)
//	//http.HandleFunc("/chat", chatokno)
//	log.Println("Слушаем на порту :8081")
//	openBrowser("http://localhost:8081/chat")
//	log.Fatal(http.ListenAndServe(":8081", nil))
//}
