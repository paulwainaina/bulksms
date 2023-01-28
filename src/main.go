package main

import (
	"fmt"
	"net/http"
	"os"
	"text/template"

	"example.com/members"
	"example.com/users"
	"example.com/session"
)

var (
	tpl *template.Template
	auth =session.NewSessionManager()
)

func init() {
	tpl = template.Must(template.ParseGlob("templates/*.html"))
}

type Page struct {
	Body  []byte
	Title string
	Data  interface{}
	Error error
}

func LoadPage(file string) (*Page, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	var body []byte
	_, err = f.Read(body)
	if err != nil {
		return nil, err
	}
	return &Page{Body: body}, nil
}
func RenderTemplate(w http.ResponseWriter, file string, page *Page) {
	err := tpl.ExecuteTemplate(w, file, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func MemberHandler(w http.ResponseWriter, r *http.Request) {
	file := "members.html"
	filePath := "templates/" + file
	pageName := "Members Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	RenderTemplate(w, file, page)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	file := "login.html"
	filePath := "templates/" + file
	pageName := "Login Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	RenderTemplate(w, file, page)
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	file := "login.html"
	filePath := "templates/" + file
	pageName := "Login Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	RenderTemplate(w, file, page)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "http://"+string(os.Getenv("SERVER")+":"+os.Getenv("SERVER_PORT")))
			w.Header().Set("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			return
		}
		if r.URL.Path != "/login" || r.URL.Path != "/loginPage" {
			cok, err := r.Cookie(os.Getenv("AuthCookieName"))
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusMovedPermanently)
				return
			}
			_,err=auth.SessionExist(cok.Value)
			if err!=nil {
				http.Redirect(w, r, "/login", http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	users := users.NewUsers(auth)
	http.Handle("/users", middleware(http.HandlerFunc(users.ServeHTTP)))
	http.Handle("/users/", middleware(http.HandlerFunc(users.ServeHTTP)))
	http.Handle("/login", middleware(http.HandlerFunc(users.ServeHTTP)))

	members := members.NewMembers()
	http.Handle("/members", middleware(http.HandlerFunc(members.ServeHTTP)))
	http.Handle("/members/", middleware(http.HandlerFunc(members.ServeHTTP)))

	http.Handle("/memberPage", middleware(http.HandlerFunc(MemberHandler)))
	http.Handle("/loginPage", middleware(http.HandlerFunc(LoginHandler)))
	http.Handle("/messagePage", middleware(http.HandlerFunc(MessageHandler)))

	err := http.ListenAndServe(string(os.Getenv("SERVER")+":"+os.Getenv("SERVER_PORT")), nil)
	if err == http.ErrServerClosed {
		fmt.Println("Backend server closed")
	} else if err != nil {
		fmt.Println("Backend server:Error occured " + err.Error())
		os.Exit(1)
	}
}
