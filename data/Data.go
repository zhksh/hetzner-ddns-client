package data

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Record struct {
	Value string `json:"value"`
	Ttl int `json:"ttl"`
	Name string `json:"name"`
	Zone_id string `json:"zone_id"`
	Zone string		`json:"zone"`
	Created string `json:"created"`
	Modified string `json:"modified"`
	Type string `json:"type"`
	Id string `json:"id"`
}

func  NewRecordPayload(Name string, Value string, Zone_id string, zone string, protocol string) *Record {
	if protocol == "udp6" {
		protocol = "AAAA"
	} else {
		protocol = "A"
	}
	return &Record{
		Name:    Name,
		Value:   Value,
		Zone_id: Zone_id,
		Zone:    zone,
		Ttl:     86400,
		Type:    protocol}
}

func (r *Record) Persist(db *sql.DB)(string){
	sql_ := fmt.Sprintf(`INSERT OR REPLACE INTO records(sub, ip4, zone_id, zone, modified, created, record_id, ttl, record_type ) 
	VALUES (?,?,?,?,?,?,?,?,?)`,
	)
	stm, err := db.Prepare(sql_)
	if err != nil {
		return fmt.Sprintf("pesisting prepare stmt failed : %v", err)
	}
	_, err = stm.Exec(r.Name, r.Value, r.Zone_id, r.Zone, r.Modified, r.Created, r.Id, r.Ttl, r.Type)
	if err != nil {
		return fmt.Sprintf("pesisting failed : %v", err)
	}

	return ""
}

func (r *Record) Delete(db *sql.DB){
	sql_ := fmt.Sprintf(`DELETE FROM records WHERE record_id = ?`,
	)

	stm, err := db.Prepare(sql_)
	stm.Exec(r.Id)
	if err != nil {
		log.Fatalf("deleting record error: %v", err)
	}
}

type RecordRequest struct {
	Subdomain string `json:"sub"`
	Zone string `json:"zone"`
	Protocol string `json:"protocol"`
}

type RequestZoneResp struct {
	Zones []struct {
		Id string `json:"id"`
		Modified string `json:"modified"`
		NS []string `json:"ns"`
	} `json:"zones"`
}

type Config struct {
	ApiKey string
	AuthHeaderKey string
	BaseUrl string
	Timeout time.Duration
}

func NewConfig(key string, baseUrl string, ahk string) (*Config){
	return &Config{
		ApiKey: key,
		BaseUrl: baseUrl,
		AuthHeaderKey: ahk,
		Timeout: 3*time.Second	}
}

type HetznerDNSApiRequest struct {
	ApiKey string
	AuthHeaderKey string
	http.Request
}

func NewHDAReq( method string, pathSuffix string, body []byte, conf Config,) *http.Request {

	httpReq, _ := http.NewRequest(method,  conf.BaseUrl + pathSuffix, bytes.NewBuffer(body))
	httpReq.Header.Add(conf.AuthHeaderKey, conf.ApiKey)

	return httpReq
}

var ReturnStatus = map[int]string{
	401 : "unauthorized",
	403 : "forbidden",
	404 : "not found",
	406 : "not accepatble",
	422 : "impossible entry"}