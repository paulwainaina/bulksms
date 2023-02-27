package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"example.com/members"
	"example.com/session"
	"example.com/users"
	"example.com/groups"
	"example.com/districts"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	tpl    *template.Template
	auth   = session.NewSessionManager()
	client *mongo.Client
	memb   *members.Members
	group *groups.Groups
	dist *districts.Districts
)

func init() {
	tpl = template.Must(template.ParseGlob("./templates/*.html"))
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
	page.Data = memb.TargetMembers
	RenderTemplate(w, file, page)
}

type BulkMessage struct {
	Numbers  []interface{} `bson:"Numbers"`
	District string        `bson:"District"`
	Title    string        `bson:"Title"`
	Message  string        `bson:"Message"`
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	bulk := BulkMessage{}
	err := json.NewDecoder(r.Body).Decode(&bulk)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ch := make(chan *http.Response)
	recp := ""
	for i := 0; i < len(bulk.Numbers); i++ {
		if recp=="" {
			recp += bulk.Numbers[i].(string)
		}else{
			recp += ","
			recp += bulk.Numbers[i].(string)
		}
	}
	if bulk.District!=""{
		n, err := strconv.ParseInt(bulk.District, 10, 64)
		if err != nil {
			res := struct{ Error string }{Error: err.Error()}
			json.NewEncoder(w).Encode(res)
			return
		}
		for _,u := range memb.TargetMembers{
			if u.District==uint(n) && !strings.Contains(recp,u.PhoneNumber){
				if recp=="" {
					recp += u.PhoneNumber
				}else{
					recp += ","
					recp += u.PhoneNumber
				}
			}
		}
	}
	if recp=="" {
		res := struct{ Error string }{Error: "No Recipients for the message"}
		json.NewEncoder(w).Encode(res)
		return
	}
	go sendasync(bulk.Message,recp, ch)
	resp := <-ch
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(bytes)
}
func sendasync(message string,to string, rc chan *http.Response) error {
	jdata := url.Values{}
	jdata.Set("username", os.Getenv("USERNAME"))		
	jdata.Set("to", to)
	jdata.Set("message", message)
	jdata.Set("from", os.Getenv("FROM"))
	res, err := http.NewRequest(http.MethodPost, os.Getenv("APIURL"), strings.NewReader(jdata.Encode()))
	if err != nil {
		return err
	}
	res.Header.Add("Accept", " Application/json")
	res.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res.Header.Add("apiKey", os.Getenv("APIKEY"))

	client := &http.Client{}
	resp, err := client.Do(res)
	if err == nil {
		rc <- resp
	}
	return err
}
func MessagePageHandler(w http.ResponseWriter, r *http.Request) {
	file := "message.html"
	filePath := "templates/" + file
	pageName := "Message Page"
	page, err := LoadPage(filePath)
	if err != nil {
		page = &Page{Title: pageName}
	}
	page.Title = pageName
	page.Data = memb.TargetMembers
	RenderTemplate(w, file, page)
}
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cok, _ := r.Cookie(os.Getenv("AuthCookieName"))
	auth.DeleteSessionByID(cok.Value)
	http.SetCookie(w, &http.Cookie{
		Name:     os.Getenv("AuthCookieName"),
		Value:    "",
		Expires:  time.Now(),
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
}
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		res := struct{ Error string }{Error: err.Error()}
		json.NewEncoder(w).Encode(res)
		return
	}
	file, _, err := r.FormFile("Passport")
	if err != nil {
		res := struct{ Error string }{Error: err.Error()}
		json.NewEncoder(w).Encode(res)
		return
	}
	defer file.Close()
	tmp, err :=  ioutil.TempFile("./assets/images", "upload-*.png")
	if err != nil {
		res := struct{ Error string }{Error: err.Error()}
		json.NewEncoder(w).Encode(res)
		return
	}
	defer tmp.Close()
	filebyte, err := ioutil.ReadAll(file)
	if err != nil {
		res := struct{ Error string }{Error: err.Error()}
		json.NewEncoder(w).Encode(res)
		return
	}
	tmp.Write(filebyte)
	json.NewEncoder(w).Encode(struct{ Path string }{Path: tmp.Name()})
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
		} else if r.URL.Path == "/registerPage" {
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
	/*p, err := os.Getwd()
	if err == nil {
		fmt.Println(p)
	}*/
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	users := users.NewUsers(auth, client)
	http.Handle("/users", middleware(http.HandlerFunc(users.ServeHTTP)))
	http.Handle("/users/", middleware(http.HandlerFunc(users.ServeHTTP)))
	http.Handle("/login", middleware(http.HandlerFunc(users.ServeHTTP)))

	memb = members.NewMembers(client)
	http.Handle("/members", middleware(http.HandlerFunc(memb.ServeHTTP)))
	http.Handle("/members/", middleware(http.HandlerFunc(memb.ServeHTTP)))

	group = groups.NewGroups(client)
	http.Handle("/groups", middleware(http.HandlerFunc(group.ServeHTTP)))
	http.Handle("/groups/", middleware(http.HandlerFunc(group.ServeHTTP)))

	dist = districts.NewDistricts(client)
	http.Handle("/districts", middleware(http.HandlerFunc(dist.ServeHTTP)))
	http.Handle("/districts/", middleware(http.HandlerFunc(dist.ServeHTTP)))

	http.Handle("/membersPage", middleware(http.HandlerFunc(MemberHandler)))
	http.Handle("/loginPage", middleware(http.HandlerFunc(LoginHandler)))
	http.Handle("/messagesPage", middleware(http.HandlerFunc(MessagePageHandler)))
	http.Handle("/index", middleware(http.HandlerFunc(IndexHandler)))
	//http.Handle("/registerPage", middleware(http.HandlerFunc(RegisterHandler)))
	http.Handle("/upload", middleware(http.HandlerFunc(UploadHandler)))
	http.Handle("/message", middleware(http.HandlerFunc(MessageHandler)))
	http.Handle("/logout", middleware(http.HandlerFunc(LogoutHandler)))

	http.Handle("/", http.RedirectHandler("/index", http.StatusSeeOther))

	err = http.ListenAndServe(string(os.Getenv("SERVER")+":"+os.Getenv("SERVER_PORT")), nil)
	if err == http.ErrServerClosed {
		fmt.Println("Backend server closed")
	} else if err != nil {
		fmt.Println("Backend server:Error occured " + err.Error())
		os.Exit(1)
	}
}
