package lib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"zhksh.io/hetzner-dnsapi-client/data"
)

type DnsService struct {
	Config data.Config
	Db *sql.DB
}
func (dns *DnsService) AddRecord(request data.RecordRequest)(*http.Response, string){
	existing_record , errstr := dns.FetchRecordBy(
		fmt.Sprintf("sub = '%s' AND zone = '%s'", request.Subdomain, request.Zone))
	if errstr != ""{
		return nil, errstr
	}
	if existing_record != nil {
		return nil, fmt.Sprintf("use update, exisiting record for %s.%s", request.Subdomain, request.Zone)
	}
	zone, errstr := dns.RequestZoneId(request.Zone)
	if errstr != "" {
		return nil, errstr
	}

	value := GetPubIP(request.Protocol)
	newRecord := data.NewRecordPayload(request.Subdomain, value, zone.Zones[0].Id, request.Zone, request.Protocol)

	response, errstr := Call("POST", "records", newRecord, &dns.Config )

	if errstr != "" {
		return nil, errstr
	}
	resonseData := map[string]data.Record{}
	body, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(body, &resonseData)
	record := resonseData["record"]
	record.Zone = request.Zone //thats an extra field in local db
	errstr = record.Persist(dns.Db)
	if errstr != "" {
		log.Printf("call successful, presist failed: %s", errstr)
		return nil, errstr
	}
	return response, ""
}


func (dns *DnsService) UpdateRecord(request data.RecordRequest)(*http.Response, string, string, bool){
	record , errstr:=  dns.FetchRecordBy(
		fmt.Sprintf("sub = '%s' AND zone = '%s'", request.Subdomain, request.Zone))
	if errstr != ""{
		return nil, "", errstr, true
	}
	if record == nil {
		return nil, fmt.Sprintf("add record first for %s.%s", request.Subdomain, request.Zone), "", true
	}
	newValue := GetPubIP(request.Protocol)
	if newValue == record.Value {
		return nil, fmt.Sprintf("no update made: old and new ip are: %s", record.Value), "", false
	}
	record.Value = newValue

	response, errstr := Call("PUT", fmt.Sprintf("records/%s", record.Id), record, &dns.Config)
	if errstr != ""{
		return nil, "", errstr, true
	}
	responseData := map[string]data.Record{}
	body, err := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, "", fmt.Sprintf("unmarshalling response : %v", err), true
	}
	record.Modified = responseData["record"].Modified
	record.Persist(dns.Db)

	return response, fmt.Sprintf("newip: %s", record.Value), "", true
}

func (dns *DnsService) DeleteRecord(request data.RecordRequest)(*http.Response, string){
	record , errstr := dns.FetchRecordBy(fmt.Sprintf("sub = '%s'", request.Subdomain))
	if errstr != ""{
		return nil, errstr
	}
	if record == nil {
		return nil, fmt.Sprintf("add record first for %s.%s", request.Subdomain, request.Zone)
	}
	response, errstr := Call("DELETE", fmt.Sprintf("records/%s", record.Id), nil, &dns.Config)
	if errstr != ""{
		return nil, errstr
	}
	record.Delete(dns.Db)

	return response, ""
}

func (dns *DnsService)RequestZoneId(zone string) (*data.RequestZoneResp, string) {
	apiResp, errstr := Call("GET", "zones?name=" + zone, nil, &dns.Config)
	if errstr != "" {
		return nil, errstr
	}
	body, err := ioutil.ReadAll(apiResp.Body)
	var zoneResp data.RequestZoneResp
	err = json.Unmarshal(body, &zoneResp)
	if err != nil {
		errstr = err.Error()
		log.Printf(errstr)
		return nil, errstr
	}

	return &zoneResp, errstr
}

func (dns *DnsService)GetIP(request data.RecordRequest) (string, string) {
	ip := GetPubIP(request.Protocol)
	return fmt.Sprintf("%s", ip), ""
}


func (dns *DnsService)FetchRecordBy(by string) (*data.Record, string){
	sql_ := fmt.Sprintf("SELECT * FROM records WHERE %s ", by)
	rows, err := dns.Db.Query(sql_)
	if err != nil {
		return nil, fmt.Sprintf("query from db: %v", err)
	}
	defer rows.Close()
	records := []data.Record{}
	for rows.Next(){
		record := data.Record{}
		id := 0
		err := rows.Scan(&id, &record.Name, &record.Value, &record.Zone_id,
			&record.Modified, &record.Created, &record.Id,  &record.Ttl , &record.Type, &record.Zone)
		if err != nil {
			return nil, fmt.Sprintf("scanning db result: %v", err)
		}
		records = append(records, record)
	}
	if len(records) > 1 {
		return nil, fmt.Sprintf("db inconstincy found %d records", len(records))
	}
	if len(records) == 1 {
		return &records[0], ""
	}
	return nil, ""
}