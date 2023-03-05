package groups

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

type Group struct {
	ID          uint   `bson:"ID"`
	Name        string `bson:"Name"`
	Description string `bson:"Description"`
	Logo        string `bson:"Logo"`
}

func (g *Group) UnmarshalJSON(data []byte) error {
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
				g.ID = uint(i)
			}
		case "name":
			{
				g.Name = string(v.(string))
			}
		case "description":
			{
				g.Description = string(v.(string))
			}
		case "logo":
			{
				g.Logo = string(v.(string))
			}
		}
	}
	return nil
}

type Groups struct {
	TargetGroups []*Group
	pattern       *regexp.Regexp
	db            *mongo.Database
}
var (
	groupCollection = "group"
)
func NewGroups(db *mongo.Database) *Groups {
	col := db.Collection(groupCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	mem := make([]*Group, 0)
	if err != nil {
		log.Fatal("Error loading groups")
	} else {
		if err = result.All(context.TODO(), &mem); err != nil {
			fmt.Println("Error parsing groups data " + err.Error())
		}
	}
	return &Groups{TargetGroups: mem, pattern: regexp.MustCompile(`^/groups/(\d+)/?`), db:db}
}

func (groups *Groups) GenerateNewID() uint {
	var x uint = 0
	if len(groups.TargetGroups) == 0 {
		return 1
	}
	var ids []uint = make([]uint, 0)
	for _, i := range groups.TargetGroups {
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

func (groups *Groups) AddGroup(memb Group) (*Group, error) {
	if memb.ID != 0 {
		return &Group{}, fmt.Errorf("new group cannot have an id %v ", memb.ID)
	}
	memb.ID = groups.GenerateNewID()
	col := groups.db.Collection(groupCollection)
	_, err := col.InsertOne(context.TODO(), memb)
	if err != nil {
		return &Group{}, err
	}
	groups.TargetGroups = append(groups.TargetGroups, &memb)
	return &memb, nil
}

func (groups *Groups) GetGroupByID(id uint) (*Group, error) {
	for _, m := range groups.TargetGroups {
		if m.ID == id {
			return m, nil
		}
	}
	return &Group{}, fmt.Errorf("group with id %v not found", id)
}

func (groups *Groups) DeleteGroupByID(id uint) (*Group, error) {
	for i, m := range groups.TargetGroups {
		if m.ID == id {
			col := groups.db.Collection(groupCollection)
			_, err := col.DeleteOne(context.TODO(), bson.M{"ID": id})
			if err != nil {
				return &Group{}, err
			}
			groups.TargetGroups = append(groups.TargetGroups[:i], groups.TargetGroups[i+1:]...)
			return m, nil
		}
	}
	return &Group{}, fmt.Errorf("group with id %v not found", id)
}

func (groups *Groups) UpdateGroup(memb Group) (*Group, error) {
	for _, m := range groups.TargetGroups {
		if m.ID == memb.ID {
			col := groups.db.Collection(groupCollection)
			_, err := col.UpdateOne(context.TODO(), bson.M{"ID": m.ID},
				bson.M{"$set": bson.M{
					"Name":        memb.Name,
					"Logo": memb.Logo,
					"Description":    memb.Description,
				}})
			if err != nil {
				return &Group{}, err
			}
			*m = memb
			return m, nil
		}
	}
	return &Group{}, fmt.Errorf("group with id %v not found", memb.ID)
}

func (groups *Groups) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/groups" {
		switch r.Method {
		case http.MethodGet:
			{
				v, err := json.Marshal(groups.TargetGroups)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(v)
			}
		case http.MethodPost:
			{
				var group Group
				err := json.NewDecoder(r.Body).Decode(&group)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				v, err := groups.AddGroup(group)
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
				var group Group
				err := json.NewDecoder(r.Body).Decode(&group)
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				v, err := groups.UpdateGroup(group)
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
		matches := groups.pattern.FindStringSubmatch(r.URL.Path)
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
				product, err := groups.GetGroupByID(uint(id))
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
				product, err := groups.DeleteGroupByID(uint(id))
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