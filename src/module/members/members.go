package members

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Member struct {
	ID          uint64 `bson:"ID"`
	Name        string `bson:"Name"`
	Gender      string `bson:"Gender"`
	DateofBirth string `bson:"DateofBirth"`
	Passport    string `bson:"Passport"`
	Deceased    bool   `bson:"Deceased"`
	PhoneNumber string `bson:"PhoneNumber"`
}

func (member *Member) UnmarshalJSON(data []byte) error {
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
		case "deceased":
			{
				member.Deceased = bool(v.(bool))
			}
		case "phonenumber":
			{
				member.PhoneNumber = string(v.(string))
			}
		case "dateofbirth":
			{
				t, err := time.Parse(time.RFC3339, string(v.(string)))
				if err != nil {
					return err
				}
				member.DateofBirth = t.String()
			}
		}
	}
	return nil
}

type Members struct {
	TargetMembers []*Member
	pattern       *regexp.Regexp
}

func NewMembers() *Members {
	return &Members{TargetMembers: make([]*Member, 0), pattern: regexp.MustCompile(`^/members/(\d+)/?`)}
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
	for _, m := range members.TargetMembers {
		if m.PhoneNumber == memb.PhoneNumber {
			return &Member{}, fmt.Errorf("a member with the same number exists %v ", m.PhoneNumber)
		}
	}
	memb.ID = members.GenerateNewID()
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
			members.TargetMembers = append(members.TargetMembers[:i], members.TargetMembers[i+1:]...)
			return m, nil
		}
	}
	return &Member{}, fmt.Errorf("member with id %v not found", id)
}

func (members *Members) UpdateMember(memb Member) (*Member, error) {
	for _, m := range members.TargetMembers {
		if m.ID == memb.ID {
			m = &memb
			return m, nil
		}
	}
	return &Member{}, fmt.Errorf("member with id %v not found", memb.ID)
}

func (members *Members) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/members" {
		switch r.Method {
		case http.MethodGet:
			{
				v, err := json.Marshal(members.TargetMembers)
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
				var member Member
				err := json.NewDecoder(r.Body).Decode(&member)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				v, err := members.AddMember(member)
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
				var member Member
				err := json.NewDecoder(r.Body).Decode(&member)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				v, err := members.UpdateMember(member)
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
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(product)
			}
		case http.MethodDelete:
			{
				product, err := members.DeleteMemberByID(uint64(id))
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
	}
}
