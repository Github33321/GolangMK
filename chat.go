package main

import (
	"fmt"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
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
func chat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
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

	senderLogin := r.FormValue("login")
	receiverLogin := r.FormValue("receiver")

	messages, err := getMessages(senderLogin, receiverLogin)
	if err != nil {
		http.Error(w, "Ошибка при получении сообщений", http.StatusInternalServerError)
		return
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

     <h2>История сообщений:</h2>
     <ul>
        {{range .Messages}}
        <li><strong>ID:</strong> {{.ID}}, <strong>Text:</strong> {{.Text}}, <strong>Time:</strong> {{.Time}}, <strong>Sender:</strong> {{.SenderLogin}}, <strong>Accepter:</strong> {{.AccepterLogin}}</li>
        {{end}}
     </ul>

    </body>
    </html>
    `

	t, err := template.New("chat").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Messages      []Message
		SenderLogin   string
		ReceiverLogin string
	}{
		Messages:      messages,
		SenderLogin:   senderLogin,
		ReceiverLogin: receiverLogin,
	}

	t.Execute(w, data)
}

//func main() {
//	initDB()
//	defer db.Close()
//	http.HandleFunc("/chat", chat)
//	//http.HandleFunc("/chat", chatokno)
//	log.Println("Слушаем на порту :8081")
//	openBrowser("http://localhost:8081/chat")
//	log.Fatal(http.ListenAndServe(":8081", nil))
//}
