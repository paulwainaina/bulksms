package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/session"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       uint64 `bson:"ID"`
	Name     string `bson:"Name"`
	Passport string `bson:"Passport"`
	Password string `bson:"Password"`
	Email    string `bson:"Email"`
}

func (user *User) UnmarshalJSON(data []byte) error {
	var jsonData map[string]interface{}
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}
	for k, v := range jsonData {
		switch strings.ToLower(k) {
		case "id":
			{
				str := string(v.(string))
				i, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					return err
				}
				user.ID = uint64(i)
			}
		case "name":
			{
				user.Name = string(v.(string))
			}
		case "password":
			{
				user.Password = string(v.(string))
			}
		case "passport":
			{
				user.Passport = string(v.(string))
			}
		case "email":
			{
				user.Email = string(v.(string))
			}
		}
	}
	return nil
}

type Users struct {
	systemUsers []*User
	pattern     *regexp.Regexp
	authSession *session.SessionManager
}

func NewUsers(auth *session.SessionManager) *Users {
	return &Users{systemUsers: make([]*User, 0), pattern: regexp.MustCompile(`^/users/(\d+)/?`), authSession: auth}
}

func (users *Users) GenerateNewID() uint64 {
	var x uint64 = 0
	if len(users.systemUsers) == 0 {
		return 1
	}
	var ids []uint64 = make([]uint64, 0)
	for _, i := range users.systemUsers {
		ids = append(ids, i.ID)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	for i := 1; i < len(ids); i++ {
		if ids[i-1]+1 != ids[i] {
			x = ids[i-1] + 1
			break
		}
	}
	if x == 0 {
		x = ids[len(ids)-1] + 1
	}
	return x
}

func (users *Users) AddUser(usr User) (*User, error) {
	if usr.ID != 0 {
		return &User{}, fmt.Errorf("new user cannot have an id %v ", usr.ID)
	}
	for _, m := range users.systemUsers {
		if m.Email == usr.Email {
			return &User{}, fmt.Errorf("a user with the same number exists %v ", m.Email)
		}
	}
	usr.ID = users.GenerateNewID()
	h, _ := bcrypt.GenerateFromPassword([]byte(usr.Password),14)
	usr.Password = string(h)
	users.systemUsers = append(users.systemUsers, &usr)
	return &usr, nil
}

func (users *Users) GetUserByID(id uint64) (*User, error) {
	for _, m := range users.systemUsers {
		if m.ID == id {
			return m, nil
		}
	}
	return &User{}, fmt.Errorf("user with id %v not found", id)
}

func (users *Users) GetUserByEmail(email string) (*User, error) {
	for _, m := range users.systemUsers {
		if m.Email == email {
			return m, nil
		}
	}
	return &User{}, fmt.Errorf("user with phone number %v not found", email)
}

func (users *Users) DeleteUserByID(id uint64) (*User, error) {
	for i, m := range users.systemUsers {
		if m.ID == id {
			users.systemUsers = append(users.systemUsers[:i], users.systemUsers[i+1:]...)
			return m, nil
		}
	}
	return &User{}, fmt.Errorf("user with id %v not found", id)
}

func (users *Users) UpdateUser(usr User) (*User, error) {
	for _, m := range users.systemUsers {
		if m.ID == usr.ID {
			m = &usr
			return m, nil
		}
	}
	return &User{}, fmt.Errorf("user with id %v not found", usr.ID)
}

func (users *Users) LoginUser(usr User) (*User, error) {
	for _, m := range users.systemUsers {
		if m.Email == usr.Email {
			err := bcrypt.CompareHashAndPassword([]byte(m.Email), []byte(usr.Password))
			if err != nil {
				return &User{}, fmt.Errorf("wrong credentials")
			}
			return m, nil
		}
	}
	return &User{}, fmt.Errorf("wrong credentials")
}
func (users *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/login" {
		w.Header().Set("Access-Control-Allow-Origin", "http://"+string(os.Getenv("SERVER")+":"+os.Getenv("SERVER_PORT")))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		u, err := users.LoginUser(user)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
			return
		}
		sess, err := users.authSession.CreateSession(u.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     os.Getenv("AuthCookieName"),
			Value:    sess.SessionID,
			Expires:  time.Now().Add(time.Minute * 20),
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		})

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(u)
		return
	}
	if r.URL.Path == "/users" {
		switch r.Method {
		case http.MethodGet:
			{
				v, err := json.Marshal(users.systemUsers)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(v)
			}
		case http.MethodPost:
			{
				var user User
				err := json.NewDecoder(r.Body).Decode(&user)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				v, err := users.AddUser(user)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(v)
			}
		case http.MethodPut:
			{
				var user User
				err := json.NewDecoder(r.Body).Decode(&user)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				v, err := users.UpdateUser(user)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(v)
			}
		default:
			{
				w.WriteHeader(http.StatusNotImplemented)
				w.Write([]byte("method not implemented"))
			}
		}
		return
	} else {
		matches := users.pattern.FindStringSubmatch(r.URL.Path)
		if len(matches) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			{
				product, err := users.GetUserByID(uint64(id))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(product)
			}
		case http.MethodDelete:
			{
				product, err := users.DeleteUserByID(uint64(id))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(product)
			}
		default:
			{
				w.WriteHeader(http.StatusNotImplemented)
				w.Write([]byte("method not implemented"))
			}
		}
		return
	}
}
