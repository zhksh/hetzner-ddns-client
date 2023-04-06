package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"zhksh.io/hetzner-dnsapi-client/data"
	"zhksh.io/myip"
)

func GetPubIP(protocol string) string {
	myip := myip.New()
	if protocol != "" {
		myip.Protocol = protocol
	}
	value, err := myip.GetPublicIP()
	if err != nil{
		log.Print(err)
	}
	return value
}

func GetPubIP6() string {
	myip := myip.New()
	myip.Protocol = "udp6"
	value, err := myip.GetPublicIP()
	if err != nil{
		log.Print(err)
	}
	return value
}

func PrintRequest(r *http.Request) []byte{
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	log.Println("Header: ")
	for key, vals := range r.Header {
		log.Printf("%s : %s", key, strings.Join(vals, ","))

	}
	log.Println("Body:")
	log.Println(string(b))
	var p data.Record
	_ = json.Unmarshal(b, &p)
	log.Println(reflect.TypeOf(p))
	log.Println(p)

	return b
}

func BodyToMap(data []byte) (map[string]interface{}, error) {
	var jsondata map[string]interface{}
	err := json.Unmarshal(data, &jsondata)

	return jsondata, err
}



func Call(method string, endpoint string, body interface{}, config *data.Config) (*http.Response, string) {
	payload, err := json.Marshal(body)
	req := data.NewHDAReq(method, endpoint, payload, *config)
	client := http.Client{Timeout: config.Timeout}
	apiResp, err := client.Do(req)
	log.Printf("Call: %s path: %s, payload: %s ", method, req.URL.Path, string(payload))
	if err != nil {
		log.Print(err)
	}
	var errstr string
	if apiResp == nil {
		errstr = "no reponse"
		log.Printf(errstr)
	} else if apiResp.StatusCode != 200 {
		errstr = fmt.Sprintf("api complaint: %s (%d)", data.ReturnStatus[apiResp.StatusCode], apiResp.StatusCode)
		log.Println(errstr)
	}
	return apiResp, errstr
}


func ContainsEmpty(ss ...string) bool {
	for _, s := range ss {
		if s == "" {
			return true
		}
	}
	return false
}