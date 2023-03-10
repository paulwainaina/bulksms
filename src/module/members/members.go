package members

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Member struct {
	ID             uint64 `bson:"ID"`
	Name           string `bson:"Name"`
	Gender         string `bson:"Gender"`
	DateofBirth    string `bson:"DateofBirth"`
	Passport       string `bson:"Passport"`
	PhoneNumber    string `bson:"PhoneNumber"`
	Email          string `bson:"Email"`
	District       uint   `bson:"District"`
	Group          []uint `bson:"Group"`
	Full           bool   `bson:"Full"`
	DateofDeath    string `bson:"DateofDeath"`
	SID            uint64 `bson:"SID"`
	DateofMarriage string `bson:"DateofMarriage"`
	DateofCatch    string `bson:"DateofCatch"`
	DateofBap      string `bson:"DateofBap"`
}

func (member *Member) Marshal(v interface{}) ([]byte, error) {
	memb, err := json.Marshal(Member{
		ID:             member.ID,
		Name:           member.Name,
		Gender:         member.Gender,
		PhoneNumber:    member.PhoneNumber,
		DateofBirth:    member.DateofBirth,
		Email:          member.Email,
		District:       member.District,
		Group:          member.Group,
		Full:           member.Full,
		DateofDeath:    member.DateofDeath,
		DateofMarriage: member.DateofMarriage,
		DateofCatch:    member.DateofCatch,
		DateofBap:      member.DateofBap,
		SID:            member.SID,
	})
	return memb, err
}

func (member *Member) UnmarshalJSON(data []byte) error {
	var jsonData map[string]interface{}
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}
	for k, v := range jsonData {
		switch strings.ToLower(k) {
		case "group":
			{
				l := make([]uint, 0)
				var vals []interface{} = v.([]interface{})
				for _, x := range vals {
					i, err := strconv.ParseInt(string(x.(string)), 10, 32)
					if err != nil {
						return err
					}
					l = append(l, uint(i))
				}
				member.Group = l
			}
		case "district":
			{
				str := string(v.(string))
				if len(str) == 0 {
					break
				}
				i, err := strconv.ParseInt(str, 10, 32)
				if err != nil {
					return err
				}
				member.District = uint(i)
			}
		case "email":
			{
				member.Email = string(v.(string))
			}
		case "dateofcatch":
			{
				member.DateofCatch = v.(string)
			}
		case "dateofbap":
			{
				member.DateofBap = v.(string)
			}
		case "dateofmarriage":
			{
				member.DateofMarriage = v.(string)
			}
		case "dateofdeath":
			{
				member.DateofDeath = v.(string)
			}
		case "full":
			{
				member.Full = v.(bool)
			}
		case "sid":
			{
				str := string(v.(string))
				if len(str) == 0 {
					break
				}
				i, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					return err
				}
				member.SID = uint64(i)
			}
		case "id":
			{
				str := string(v.(string))
				if len(str) == 0 {
					break
				}
				i, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					return err
				}
				member.ID = uint64(i)
			}
		case "name":
			{
				member.Name = string(v.(string))
			}
		case "gender":
			{
				member.Gender = string(v.(string))
			}
		case "passport":
			{
				member.Passport = string(v.(string))
			}
		case "phonenumber":
			{
				member.PhoneNumber = string(v.(string))
			}
		case "dateofbirth":
			{
				member.DateofBirth = v.(string)
			}

		}
	}
	return nil
}

type Members struct {
	TargetMembers []*Member
	pattern       *regexp.Regexp
	db            *mongo.Database
}

var (
	memberCollection = "member"
)

func NewMembers(db *mongo.Database) *Members {
	col := db.Collection(memberCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	mem := make([]*Member, 0)
	if err != nil {
		log.Fatal("Error loading members")
	} else {
		if err = result.All(context.TODO(), &mem); err != nil {
			fmt.Println("Error parsing purchases data " + err.Error())
		}
	}
	return &Members{TargetMembers: mem, pattern: regexp.MustCompile(`^/members/(\d+)/?`), db: db}
}

func (members *Members) GenerateNewID() uint64 {
	var x uint64 = 0
	if len(members.TargetMembers) == 0 {
		return 1
	}
	var ids []uint64 = make([]uint64, 0)
	for _, i := range members.TargetMembers {
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

func (members *Members) AddMember(memb Member) (*Member, error) {
	if memb.ID != 0 {
		return &Member{}, fmt.Errorf("new member cannot have an id %v ", memb.ID)
	}
	if memb.PhoneNumber == "" {
		return &Member{}, fmt.Errorf("new member has to have a phone number %v ", memb.ID)
	}
	for _, m := range members.TargetMembers {
		if m.PhoneNumber == memb.PhoneNumber {
			return &Member{}, fmt.Errorf("a member with the same number exists %v ", m.PhoneNumber)
		}
	}
	memb.ID = members.GenerateNewID()
	col := members.db.Collection(memberCollection)
	_, err := col.InsertOne(context.TODO(), memb)
	if err != nil {
		return &Member{}, err
	}
	members.TargetMembers = append(members.TargetMembers, &memb)
	return &memb, nil
}

func (members *Members) GetMemberByID(id uint64) (*Member, error) {
	for _, m := range members.TargetMembers {
		if m.ID == id {
			return m, nil
		}
	}
	return &Member{}, fmt.Errorf("member with id %v not found", id)
}

func (members *Members) GetMemberByPhone(phonenumber string) (*Member, error) {
	for _, m := range members.TargetMembers {
		if m.PhoneNumber == phonenumber {
			return m, nil
		}
	}
	return &Member{}, fmt.Errorf("member with phone number %v not found", phonenumber)
}

func (members *Members) DeleteMemberByID(id uint64) (*Member, error) {
	for i, m := range members.TargetMembers {
		if m.ID == id {
			col := members.db.Collection(memberCollection)
			_, err := col.DeleteOne(context.TODO(), bson.M{"ID": id})
			if err != nil {
				return &Member{}, err
			}
			members.TargetMembers = append(members.TargetMembers[:i], members.TargetMembers[i+1:]...)
			return m, nil
		}
	}
	return &Member{}, fmt.Errorf("member with id %v not found", id)
}

func (members *Members) UpdateMember(memb Member) (*Member, error) {
	for _, m := range members.TargetMembers {
		if m.ID == memb.ID {
			col := members.db.Collection(memberCollection)
			_, err := col.UpdateOne(context.TODO(), bson.M{"ID": m.ID},
				bson.M{"$set": bson.M{
					"Name":           memb.Name,
					"DateofBirth":    memb.DateofBirth,
					"Gender":         memb.Gender,
					"PhoneNumber":    memb.PhoneNumber,
					"Email":          memb.Email,
					"District":       memb.District,
					"Passport":       memb.Passport,
					"Group":          memb.Group,
					"Full":           memb.Full,
					"DateofDeath":    memb.DateofDeath,
					"SID":            memb.SID,
					"DateofMarriage": memb.DateofMarriage,
					"DateofCatch":    memb.DateofCatch,
					"DateofBap":      memb.DateofBap,
				}})
			if err != nil {
				return &Member{}, err
			}
			*m = memb
			return m, nil
		}
	}
	return &Member{}, fmt.Errorf("member with id %v not found", memb.ID)
}

func (members *Members) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path=="/searchmembers"{
		switch r.Method {
			case http.MethodPost:{
				type Search struct{
					Name string
					District []string
					Group []string
				}
				var s Search
				err:=json.NewDecoder(r.Body).Decode(&s)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				
				res:=make([]Member,0)
				for _,m:= range members.TargetMembers{
					res = append(res,*m)
				}
				if s.Name!=""{
					for i:=0;i<len(res);i++{
						if !strings.Contains(res[i].Name,s.Name) && !strings.Contains(res[i].PhoneNumber,s.Name){
							res=append(res[:i],res[i+1:]...)
							i--
						}
					}
				}
				if len(s.District)!=0{
					for _,d:=range s.District{
						x,_:=strconv.ParseInt(d,10,64)
						for i:=0;i<len(res);i++{
							if res[i].District!=uint(x) {
								res=append(res[:i],res[i+1:]... )
								i--
							}
						}
					}
				}
				if len(s.Group)!=0{
					for _,d:=range s.Group{
						x,_:=strconv.ParseInt(d,10,64)
						for i:=0;i<len(res);i++{
							var contain=false
							for _,g:=range res[i].Group{
								if g==uint(x){
									contain=true
									break
								}
							}
							if !contain{
								res=append(res[:i],res[i+1:]... )
								i--
							}
						}
					}
				}
				json.NewEncoder(w).Encode(res)
				return
			}
			default:{
				w.WriteHeader(http.StatusNotImplemented)
				w.Write([]byte("method not implemented"))
			}
		}
		return
	}
	if r.URL.Path == "/members" {
		switch r.Method {
		case http.MethodGet:
			{
				v, err := json.Marshal(members.TargetMembers)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(v)
			}
		case http.MethodPost:
			{
				var member Member
				err := json.NewDecoder(r.Body).Decode(&member)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				v, err := members.AddMember(member)
				if err != nil {
					res := struct{ Error string }{Error: err.Error()}
					json.NewEncoder(w).Encode(res)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(v)
			}
		case http.MethodPut:
			{
				var member Member
				err := json.NewDecoder(r.Body).Decode(&member)
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				v, err := members.UpdateMember(member)
				if err != nil {
					res := struct{ Error string }{Error: err.Error()}
					json.NewEncoder(w).Encode(res)
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
	} else {
		matches := members.pattern.FindStringSubmatch(r.URL.Path)
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
				product, err := members.GetMemberByID(uint64(id))
				if err != nil {
					res := struct{ Error string }{Error: err.Error()}
					json.NewEncoder(w).Encode(res)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(product)
			}
		case http.MethodDelete:
			{
				product, err := members.DeleteMemberByID(uint64(id))
				if err != nil {
					res := struct{ Error string }{Error: err.Error()}
					json.NewEncoder(w).Encode(res)
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
	}
}
