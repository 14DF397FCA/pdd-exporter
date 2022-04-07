package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Record struct {
	RecordID  int    `json:"record_id"`
	TTL       int    `json:"ttl"`
	Content   string `json:"content"`
	Domain    string `json:"domain"`
	Fqdn      string `json:"fqdn"`
	Subdomain string `json:"subdomain"`
	Type      string `json:"type"`
	Priority  string `json:"priority"`
}

type PDDRecords struct {
	Domain  string   `json:"domain"`
	Records []Record `json:"records"`
	Error   string   `json:"error,omitempty"`
	Success string   `json:"success"`
}

func ExtractBinPath() (string, error) {
	a, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(a), nil
}

func GetFilename() (string, error) {
	p, e := ExtractBinPath()
	if e != nil {
		return "", e
	}
	return p + "/pdd-exporter-" + time.Now().Format("2006-01-02_15-04-05"), nil
}

func ReadEnv(key string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return value, true
	} else {
		return "", false
	}
}

func GetToken() (string, bool) {
	if v, e := ReadEnv("PDD_TOKEN"); e {
		return v, true
	} else {
		log.Printf("Variable PDD_TOKEN not set")
		return "", false
	}
}

func GetDomain() (string, bool) {
	if v, e := ReadEnv("PDD_DOMAIN"); e {
		return v, true
	} else {
		log.Printf("Variable PDD_DOMAIN not set")
		return "", false
	}
}

func main() {
	var PDDToken, PDDDomain string

	PDDToken, b := GetToken()
	if !b {
		flag.StringVar(&PDDToken, "t", "", "Specify PDD Token for export domain's records")
	}

	PDDDomain, b = GetDomain()
	if !b {
		flag.StringVar(&PDDDomain, "d", "", "Specify domain for export")
	}

	flag.Parse()

	if len(PDDToken) == 0 {
		log.Println("Specify PDD Token for export domain's records")
		os.Exit(1)
	}
	if len(PDDDomain) == 0 {
		log.Println("Setup domain name for export")
		os.Exit(2)
	}

	url := fmt.Sprintf("https://pddimp.yandex.ru/api2/admin/dns/list?domain=%s", PDDDomain)
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Println(err)
	}

	req.Header = http.Header{
		"PddToken": []string{PDDToken},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Println(resp.Status)
	}
	pdd := PDDRecords{}
	json.Unmarshal(bodyBytes, &pdd)

	if pdd.Success != "ok" {
		log.Fatalf("An error occurred: %s", pdd.Error)
	}

	log.Printf("Exporting records from Yandex.Connect for domain: %s", pdd.Domain)

	fileName, _ := GetFilename()
	f, err := os.Create(fileName)
	defer f.Close()
	if err != nil {
		log.Println(err)
	}
	f.WriteString("domain,fqdn,subdomain,type,content,priority,ttl\n")
	for _, r := range pdd.Records {
		f.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n", r.Domain, r.Fqdn, r.Subdomain, r.Type, r.Content, r.Priority, strconv.Itoa(r.TTL)))
	}
	log.Printf("Result file: %s", fileName)
}
