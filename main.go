package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	_ "github.com/lib/pq"
)

type User struct {
	UserID    int
	UserLogin string
	UserPassw string
}

var db *sql.DB

func getNextUserID() (int, error) {
	var nextID int
	query := "SELECT COALESCE(MAX(user_id), 0) + 1 FROM users"
	err := db.QueryRow(query).Scan(&nextID)
	return nextID, err
}

func userExists(userlogin string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE userlogin = $1)"
	err := db.QueryRow(query, userlogin).Scan(&exists)
	return exists, err
}
func registerUser(user User) error {
	exists, err := userExists(user.UserLogin)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("пользователь с логином %s уже существует", user.UserLogin)
	}

	nextID, err := getNextUserID()
	if err != nil {
		return err
	}

	query := "INSERT INTO users (user_id, userlogin, userpassw) VALUES ($1, $2, $3)"
	_, err = db.Exec(query, nextID, user.UserLogin, user.UserPassw)
	return err
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.FormValue("action") == "register" {

			user := User{
				UserID:    1,
				UserLogin: r.FormValue("login"),
				UserPassw: r.FormValue("password"),
			}

			// Регистрация пользователя
			err := registerUser(user)
			if err != nil {
				http.Error(w, "Ошибка при регистрации", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "Пользователь %s успешно зарегистрирован", user.UserLogin)
			openBrowser("http://localhost:8081/chat")
		} else if r.FormValue("action") == "login" {

			login := r.FormValue("login")
			password := r.FormValue("password")

			err := authorizeUser(login, password)
			if err != nil {
				http.Error(w, "Ошибка при авторизации", http.StatusUnauthorized)
				return
			}
			fmt.Fprintf(w, "Пользователь %s успешно авторизован", login)
			openBrowser("http://localhost:8081/chat")
		}
	} else {
		// Отображение HTML формы
		tmpl := `
  <!DOCTYPE html>
  <html>
  <head>
   <title>Регистрация и Авторизация</title>
  </head>
  <body>
   <h1>Регистрация</h1>
   <form method="POST">
    <label for="login">Логин:</label>
    <input type="text" id="login" name="login" required><br><br>
    <label for="password">Пароль:</label>
    <input type="password" id="password" name="password" required><br><br>
    <input type="hidden" name="action" value="register">
    <input type="submit" value="Зарегистрироваться">
   </form>
   <h1>Авторизация</h1>
   <form method="POST">
    <label for="login">Логин:</label>
    <input type="text" id="login" name="login" required><br><br>
    <label for="password">Пароль:</label>
    <input type="password" id="password" name="password" required><br><br>
    <input type="hidden" name="action" value="login">
    <input type="submit" value="Зайти">
   </form>
  </body>
  </html>
  `

		t, err := template.New("registration").Parse(tmpl)
		if err != nil {
			log.Fatal(err)
		}
		t.Execute(w, nil)
	}
}
func authorizeUser(login, password string) error {

	user, err := getUserByLogin(login)
	if err != nil {
		return err
	}

	if user == nil {
		return fmt.Errorf("пользователь с логином %s не найден", login)
	}
	if user.UserPassw != password {
		return fmt.Errorf("неправильный пароль для пользователя %s", login)
	}

	return nil
}

func getUserByLogin(login string) (*User, error) {
	var user User

	query := "SELECT user_id, userlogin, userpassw FROM users WHERE userlogin = $1"
	row := db.QueryRow(query, login)

	err := row.Scan(&user.UserID, &user.UserLogin, &user.UserPassw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// Инициализация базы данных
func initDB() {
	var err error
	connStr := "user=rrrr password=qwerty dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()
	http.HandleFunc("/chat", chat)
	http.HandleFunc("/register", registrationHandler)

	log.Println("Слушаем на порту :8081")
	openBrowser("http://localhost:8081/register")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// Функция для открытия браузера
func openBrowser(url string) {
	var err error
	switch {
	case "windows" == runtime.GOOS:
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin" == runtime.GOOS:
		err = exec.Command("open", url).Start()
	default: // linux
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		log.Fatal(err)
	}
}
