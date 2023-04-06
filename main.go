package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"modernc.org/sqlite"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"path/filepath"
	"sync"
	"syscall"
	"zhksh.io/hetzner-dnsapi-client/data"
	"zhksh.io/hetzner-dnsapi-client/lib"
)




var mutex = &sync.Mutex{}
var db *sql.DB
var dnsService lib.DnsService
var logFilePtr *os.File
var defaultPort = "8081"
var dbFile = "db.sqlite"
var apiKey = ""
var authHeaderField = "Auth-Api-Token"

var baseApiUrl = "https://dns.hetzner.com/api/v1/"
var config = data.NewConfig(apiKey,baseApiUrl, authHeaderField)
var rootPath = getPath()

func init_() {
	var err error

	db, err = sql.Open("sqlite", path.Join(rootPath,dbFile))

	if err != nil{
		log.Println("connecting to db failed")
		panic(err)

	}
	dnsService = lib.DnsService{Config: *config, Db: db}
}


func shutDown(srv http.Server){
	log.Printf("init shutdown")
	srv.Shutdown(context.Background())
	log.Println("closing db")
	db.Close()
}

func printPostBody(w http.ResponseWriter, r *http.Request){
	lib.PrintRequest(r)
}


func getPubIp(w http.ResponseWriter, r *http.Request) {
	 data := make(map[string]string)
	 data["ip"] = lib.GetPubIP("")
	 json, _ := json.Marshal(data)
	 w.Write(json)
}

func updateRecord(w http.ResponseWriter, r *http.Request){
	var request data.RecordRequest
	retData:= map[string]string{}
	retData["success"] = "true"
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &request)
	if  err != nil {
		retData["success"] = "false"
		retData["msg"] = err.Error()
	} else {
		_ , msg, errstr, _  := dnsService.UpdateRecord(request)
		if  errstr != "" {
			retData["success"] = "false"
			retData["msg"] = errstr
		}
		if msg != "" {
			retData["msg"] = msg
		}
	}
	json, _ := json.Marshal(retData)
	w.Write(json)
}


func addRecord(w http.ResponseWriter, r *http.Request){
	var addRecReq data.RecordRequest
	retData:= map[string]string{}
	retData["success"] = "true"
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &addRecReq)
	if  err != nil {
		retData["success"] = "false"
		retData["msg"] = err.Error()
	} else {
		_, errstr := dnsService.AddRecord(addRecReq)
		if  errstr != "" {
			retData["success"] = "false"
			retData["msg"] = errstr
		}
	}

	json, _ := json.Marshal(retData)
	w.Write(json)
}

func deleteRecord(w http.ResponseWriter, r *http.Request){
	var request data.RecordRequest
	retData:= map[string]string{}
	retData["success"] = "true"
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &request)
	if  err != nil {
		retData["success"] = "false"
		retData["msg"] = err.Error()
	} else {
		_, errstr := dnsService.DeleteRecord(request)
		if  errstr != "" {
			retData["success"] = "false"
			retData["msg"] = errstr
		}
	}
	json, _ := json.Marshal(retData)
	w.Write(json)
}


func startServer(port string){
	log.Printf("starting up on port %s", port)
	logFilePtr, err := os.OpenFile(path.Join(rootPath,"log.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil{
		log.Println("getting a jandle of logfile failed failed")
		panic(err)
	}
	defer logFilePtr.Close()
	log.SetOutput(logFilePtr)

	var srv http.Server
	srv.Addr = ":" + port

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc,
			os.Interrupt,
			syscall.SIGTERM,
			syscall.SIGABRT,
			syscall.SIGHUP,
			syscall.SIGQUIT)
		s := <-sigc
		log.Printf("recieved signal %s", s)
		shutDown(srv)
	}()

	http.HandleFunc("/api/v1/pubip", getPubIp)
	http.HandleFunc("/api/v1/add-record", addRecord)
	http.HandleFunc("/api/v1/update-record", updateRecord)
	http.HandleFunc("/api/v1/delete-record", deleteRecord)



	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("shutdown : %v", err)
	}
}

func main() {
	init_()

	zone := flag.String("z", "", "zone")
	subdomain := flag.String("d", "", "subdomain")
	action := flag.String("a", "", "add update delete getip")
	port := flag.String("p", defaultPort, "port")
	protocol := flag.String("ipv", "udp4", "protocol for checkting pub ip")
	server := flag.Bool("server", false, "start as webserver")
	flag.Parse()

	if *server {
		startServer(*port)
		os.Exit(0)
	}

	var errstr string
	var msg string
	printResponse := true
	if *action == "getip" {
		request := data.RecordRequest{Protocol: *protocol}
		msg, errstr = dnsService.GetIP(request)
	} else if !lib.ContainsEmpty(*zone, *subdomain, *action){
		switch *action {
		case "add":
			request := data.RecordRequest{Zone: *zone, Subdomain: *subdomain, Protocol: *protocol}
			_, errstr = dnsService.AddRecord(request)
		case "update":
			request := data.RecordRequest{Zone: *zone, Subdomain: *subdomain, Protocol: *protocol}
			_, msg, errstr, printResponse = dnsService.UpdateRecord(request)
		case "delete":
			request := data.RecordRequest{Zone: *zone, Subdomain: *subdomain, Protocol: *protocol}
			_, errstr = dnsService.DeleteRecord(request)
		}

	}
	if errstr != ""{
		log.Printf(errstr)
	} else {
		if printResponse {
			msgResult := "success"
			if msg != ""{
				msgResult = fmt.Sprintf("%s : %s", msgResult, msg)
			}
			log.Printf(msgResult)
		}
	}




}

func getPath() string{
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}
