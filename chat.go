package main

import (
	"fmt"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
)

func sendMessage(sender, receiver, message string) error {
	if sender == "" || receiver == "" || message == "" {
		return fmt.Errorf("sender, receiver, and message cannot be empty")
	}

	query := `
        INSERT INTO public.chat (text, "time", "senderLogin", "accepterLogin")
        VALUES ( $1, NOW(), $2, $3)
    `

	_, err := db.Exec(query, message, sender, receiver)
	if err != nil {
		return fmt.Errorf("error while sending message: %v", err)
	}

	return nil
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
			http.Error(w, "Ошибка при отправке сообщения", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Сообщение от %s к %s успешно отправлено", senderLogin, receiverLogin)
	} else {
		tmpl := `
            <!DOCTYPE html>
            <html>
            <head>
                <title>Чат</title>
            </head>
            <body>
                <h1>Чат</h1>
                <form method="POST">
                    <label for="login">Логин отправителя:</label>
                    <input type="text" id="login" name="login" required><br><br>
                    <label for="receiver">Логин получателя:</label>
                    <input type="text" id="receiver" name="receiver" required><br><br>
                    <label for="mess">Сообщение:</label>
                    <input type="text" id="mess" name="mess" required><br><br>
                    <input type="submit" value="Отправить">
                </form>
            </body>
            </html>
        `
		t, err := template.New("chat").Parse(tmpl)
		if err != nil {
			log.Fatal(err)
		}
		t.Execute(w, nil)
	}
}

//func main() {
//	initDB()
//	defer db.Close()
//	http.HandleFunc("/chat", chat)
//	log.Println("Слушаем на порту :8081")
//	openBrowser("http://localhost:8081/chat")
//	log.Fatal(http.ListenAndServe(":8081", nil))
//}
