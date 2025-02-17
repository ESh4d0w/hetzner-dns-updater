package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Token       string
	Zone_Name   string
	Record_Name string
	Minutes     int
}

type Zones struct {
	Zones []Zone
}

type Zone struct {
	Id   string
	Name string
}

type RecordsList struct {
	Records []Record
}

type Records struct {
	Record Record
}

type Record struct {
	Id       string
	Name     string
	Value    string
	Zone_Id  string
	Created  string //TODO: Date
	Modified string //TODO Date
}

func sendGetZone(token string, zone_name string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://dns.hetzner.com/api/v1/zones", nil)
	req.Header.Add("Auth-API-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failure on making request %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var respJson Zones
	err = json.Unmarshal(respBody, &respJson)
	if err != nil {
		return "", fmt.Errorf("Failure on Unmarshal %w", err)
	}

	for _, element := range respJson.Zones {
		if element.Name == zone_name {
			return element.Id, nil
		}
	}
	return "", fmt.Errorf("Failure : Can't match %s to found zones", zone_name)
}

func sendGetRecord(token string, zone_id string, record_name string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://dns.hetzner.com/api/v1/records?zone_id="+zone_id, nil)
	req.Header.Add("Auth-API-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failure on making request %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var respJson RecordsList
	err = json.Unmarshal(respBody, &respJson)
	if err != nil {
		return "", fmt.Errorf("Failure on Unmarshal %w", err)
	}

	for _, element := range respJson.Records {
		if element.Name == record_name {
			return element.Id, nil
		}
	}
	return "", fmt.Errorf("Failure : Found no records matching record_name: %s", record_name)
}

func sendPutRecord(token string, zone_id string, record_id string, record_name string, record_ip string) (string, string, string, error) {
	jsonString := `{"value": "` + record_ip + `","ttl": 43200,"type": "A","name": "` + record_name + `","zone_id": "` + zone_id + `"}`
	sendjson := []byte(jsonString)
	body := bytes.NewBuffer(sendjson)

	client := &http.Client{}
	req, _ := http.NewRequest("PUT", "https://dns.hetzner.com/api/v1/records/"+record_id, body)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Auth-API-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("Failure on making request %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var respJson Records
	err = json.Unmarshal(respBody, &respJson)
	if err != nil {
		return "", "", "", fmt.Errorf("Failure on Unmarshal %w", err)
	}
	return respJson.Record.Name, respJson.Record.Value, respJson.Record.Modified, err

}

func sendGetIp() (string, error) {
	resp, err := http.Get("https://checkip.amazonaws.com")
	if err != nil {
		return "", fmt.Errorf("Failure on Get Request %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failure on ReadBody %w", err)
	}
	respString := strings.TrimSuffix(string(respBody), "\n")

	return respString, nil
}

func sendGetIpRetry() string {
	new_record, err := sendGetIp()
	if err != nil {
		log.Printf("%v", err)
		log.Println("Waiting 10 Minutes till retry")
		time.Sleep(10 * time.Minute)
		new_record, err = sendGetIp()
		if err != nil {
			log.Fatalf("Can't get IP check internet connection %v", err)
		}
	}
	return new_record
}

func loop(mins int, token string, zone_id string, record_id string, record_name string, record_ip string) {
	record_wan_ip := sendGetIpRetry()
	if record_ip != record_wan_ip {
		log.Printf("Updating...\n")
		resp_name, resp_value, resp_mod, err := sendPutRecord(token, zone_id, record_id, record_name, record_wan_ip)
		if err != nil {
			log.Fatalf("sendPutRecord failed: %v", err)
		}
		log.Printf("Update Sucessfull %s to %s", resp_name, resp_value)
		log.Printf("Was last modified at %s", resp_mod)
	}

	time.Sleep(time.Duration(mins) * time.Minute)
	loop(mins, token, zone_id, record_id, record_name, record_wan_ip)
}

func main() {
	os.Stdin.Close()

	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Read Config Failed")
	}
	var config Config
	err = json.Unmarshal(data, &config)

	log.Printf("Loaded Config\n")

	zone_id, err := sendGetZone(config.Token, config.Zone_Name)
	if err != nil {
		log.Fatalf("sendGetZone failed: %v", err)
	}

	record_id, err := sendGetRecord(config.Token, zone_id, config.Record_Name)
	if err != nil {
		log.Fatalf("sendGetRecord failed: %v", err)
	}

	loop(config.Minutes, config.Token, zone_id, record_id, config.Record_Name, "")

}
