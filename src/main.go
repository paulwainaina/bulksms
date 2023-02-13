package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"example.com/members"
	"example.com/session"
	"example.com/users"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	tpl    *template.Template
	auth   = session.NewSessionManager()
	client *mongo.Client
	err    error
	memb   *members.Members
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
	page.Data = memb.TargetMembers
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

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	file := "register.html"
	filePath := "templates/" + file
	pageName := "Register Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	RenderTemplate(w, file, page)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	file := "index.html"
	filePath := "templates/" + file
	pageName := "Home Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	page.Data=memb.TargetMembers
	RenderTemplate(w, file, page)
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	file := "message.html"
	filePath := "templates/" + file
	pageName := "Message Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	page.Data=memb.TargetMembers
	RenderTemplate(w, file, page)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "http://"+string(os.Getenv("SERVER")+":"+os.Getenv("SERVER_PORT")))
			w.Header().Set("Access-Control-Allow-Methods", "POST,GET,PUT,DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			return
		}
		if r.URL.Path == "/login" {
		} else if r.URL.Path == "/loginPage" {
		}else if r.URL.Path=="/registerPage"{
		} else {
			cok, err := r.Cookie(os.Getenv("AuthCookieName"))
			if err != nil {
				http.Redirect(w, r, "/loginPage", http.StatusMovedPermanently)
				return
			}
			_, err = auth.SessionExist(cok.Value)
			if err != nil {
				http.Redirect(w, r, "/loginPage", http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading the .env file")
	}
	client, err = mongo.NewClient(options.Client().ApplyURI(string("mongodb://" + os.Getenv("MONGO_HOST") + ":" + os.Getenv("MONGO_PORT"))))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.TODO(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	users := users.NewUsers(auth, client)
	http.Handle("/users",middleware( http.HandlerFunc(users.ServeHTTP)))
	http.Handle("/users/",middleware( http.HandlerFunc(users.ServeHTTP)))
	http.Handle("/login", middleware(http.HandlerFunc(users.ServeHTTP)))

	memb = members.NewMembers(client)
	http.Handle("/members", middleware(http.HandlerFunc(memb.ServeHTTP)))
	http.Handle("/members/", middleware(http.HandlerFunc(memb.ServeHTTP)))

	http.Handle("/membersPage", middleware(http.HandlerFunc(MemberHandler)))
	http.Handle("/loginPage", middleware(http.HandlerFunc(LoginHandler)))
	http.Handle("/messagesPage", middleware(http.HandlerFunc(MessageHandler)))
	http.Handle("/index", middleware(http.HandlerFunc(IndexHandler)))
	http.Handle("/registerPage", middleware(http.HandlerFunc(RegisterHandler)))

	http.Handle("/", http.RedirectHandler("/index", http.StatusSeeOther))

	err = http.ListenAndServe(string(os.Getenv("SERVER")+":"+os.Getenv("SERVER_PORT")), nil)
	if err == http.ErrServerClosed {
		fmt.Println("Backend server closed")
	} else if err != nil {
		fmt.Println("Backend server:Error occured " + err.Error())
		os.Exit(1)
	}
}
