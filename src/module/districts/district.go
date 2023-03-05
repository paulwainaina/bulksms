package districts

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

type District struct {
	ID          uint   `bson:"ID"`
	Name        string `bson:"Name"`
	Description string `bson:"Description"`
	Logo        string `bson:"Logo"`
}

func (d *District) UnmarshalJSON(data []byte) error {
	var jsonData map[string]interface{}
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}
	for k, v := range jsonData {
		switch strings.ToLower(k) {
		case "id":{
			str := string(v.(string))
				i, err := strconv.ParseInt(str, 10, 64)
				if err != nil {
					return err
				}
				d.ID = uint(i)
		}
	case "name":{
		d.Name=string(v.(string))
	}
	case "description":{
		d.Description=string(v.(string))
	}
	case "logo":{
		d.Logo=string(v.(string))
	}
		}
	}
	return nil
}

type Districts struct {
	TargetDistricts []*District
	pattern       *regexp.Regexp
	db            *mongo.Database
}
var (
	districtCollection = "district"
)
func NewDistricts(db *mongo.Database) *Districts {
	col := db.Collection(districtCollection)
	result, err := col.Find(context.TODO(), bson.M{})
	mem := make([]*District, 0)
	if err != nil {
		log.Fatal("Error loading districts")
	} else {
		if err = result.All(context.TODO(), &mem); err != nil {
			fmt.Println("Error parsing districts data " + err.Error())
		}
	}
	return &Districts{TargetDistricts: mem, pattern: regexp.MustCompile(`^/districts/(\d+)/?`), db: db}
}

func (districts *Districts) GenerateNewID() uint {
	var x uint = 0
	if len(districts.TargetDistricts) == 0 {
		return 1
	}
	var ids []uint = make([]uint, 0)
	for _, i := range districts.TargetDistricts {
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

func (districts *Districts) AddDistrict(memb District) (*District, error) {
	if memb.ID != 0 {
		return &District{}, fmt.Errorf("new district cannot have an id %v ", memb.ID)
	}
	memb.ID = districts.GenerateNewID()
	col := districts.db.Collection(districtCollection)
	_, err := col.InsertOne(context.TODO(), memb)
	if err != nil {
		return &District{}, err
	}
	districts.TargetDistricts = append(districts.TargetDistricts, &memb)
	return &memb, nil
}

func (districts *Districts) GetDistrictByID(id uint) (*District, error) {
	for _, m := range districts.TargetDistricts {
		if m.ID == id {
			return m, nil
		}
	}
	return &District{}, fmt.Errorf("district with id %v not found", id)
}

func (districts *Districts) DeleteDistrictByID(id uint) (*District, error) {
	for i, m := range districts.TargetDistricts {
		if m.ID == id {
			col := districts.db.Collection(districtCollection)
			_, err := col.DeleteOne(context.TODO(), bson.M{"ID": id})
			if err != nil {
				return &District{}, err
			}
			districts.TargetDistricts = append(districts.TargetDistricts[:i], districts.TargetDistricts[i+1:]...)
			return m, nil
		}
	}
	return &District{}, fmt.Errorf("district with id %v not found", id)
}

func (districts *Districts) UpdateDistrict(memb District) (*District, error) {
	for _, m := range districts.TargetDistricts {
		if m.ID == memb.ID {
			col := districts.db.Collection(districtCollection)
			_, err := col.UpdateOne(context.TODO(), bson.M{"ID": m.ID},
				bson.M{"$set": bson.M{
					"Name":        memb.Name,
					"Logo": memb.Logo,
					"Description":    memb.Description,
				}})
			if err != nil {
				return &District{}, err
			}
			*m = memb
			return m, nil
		}
	}
	return &District{}, fmt.Errorf("district with id %v not found", memb.ID)
}

func (districts *Districts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/districts" {
		switch r.Method {
		case http.MethodGet:
			{
				v, err := json.Marshal(districts.TargetDistricts)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(v)
			}
		case http.MethodPost:
			{
				var district District
				err := json.NewDecoder(r.Body).Decode(&district)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				v, err := districts.AddDistrict(district)
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
				var district District
				err := json.NewDecoder(r.Body).Decode(&district)
				if err != nil {
					fmt.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				v, err := districts.UpdateDistrict(district)
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
		matches := districts.pattern.FindStringSubmatch(r.URL.Path)
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
				product, err := districts.GetDistrictByID(uint(id))
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
				product, err := districts.DeleteDistrictByID(uint(id))
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
