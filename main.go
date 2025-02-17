package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Token         string             `yaml:"token"`
	Zone_Name     string             `yaml:"zone_name"`
	Record_Name   string             `yaml:"record_name"`
	Record_Type   string             `yaml:"record_type"`
	Change_Record ConfigChangeRecord `yaml:"change_record"`
}

type ConfigChangeRecord struct {
	Change_Name           bool   `yaml:"change_name"`
	New_Name              string `yaml:"new_name"`
	Change_Type           bool   `yaml:"change_type"`
	New_Type              string `yaml:"new_type"`
	Change_Value          bool   `yaml:"change_value"`
	Change_Value_To_Wanip bool   `yaml:"change_value_to_wanip"`
	New_Value             string `yaml:"new_value"`
}

type Zones struct {
	Zones []Zone
	Meta  Pagination
}

type Pagination struct {
	Page          string
	Per_Page      string
	Previous_Page string
	Next_Page     string
	Last_Page     string
	Total_Entries string
}

type Zone struct {
	Id               string
	Name             string
	Ttl              int
	Registrar        string
	Legacy_Dns_Host  string
	Legacy_Ns        []string
	Ns               []string
	Created          string //TODO: Date
	Verified         string
	Modified         string //TODO: Date
	Project          string
	Owner            string
	Permission       string
	Zone_Type        ZoneType
	Status           string
	Paused           bool
	Is_Secondary_Dns bool
	Txt_Verification Verification
	Records_Count    int
}

type ZoneType struct {
	Id          string
	Name        string
	Description string
	Prices      string
}

type Verification struct {
	Name  string
	Token string
}

type Records struct {
	Records []Record
}

type Record struct {
	Id       string
	Type     string
	Name     string
	Value    string
	Zone_Id  string
	Created  string //TODO: Date
	Modified string //TODO Date
}

func sendGetZone(token string, zone_name string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://dns.hetzner.com/api/v1/zones", nil)
	req.Header.Add("Auth-API-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failure sendGetZone : Making Request %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	respBody, _ := io.ReadAll(resp.Body)
	var respJson Zones
	err = json.Unmarshal(respBody, &respJson)
	if err != nil {
		return "", fmt.Errorf("Failure sendGetZone : Unmarshal failed %w", err)
	}

	for _, element := range respJson.Zones {
		if element.Name == zone_name {
			return element.Id, nil
		}
	}
	return "", fmt.Errorf("Failure : Can't match %s to found zones", zone_name)
}

func sendGetRecord(token string, zone_id string, record_name string, record_type string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://dns.hetzner.com/api/v1/records?zone_id="+zone_id, nil)
	req.Header.Add("Auth-API-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failure sendGetRecord : Making Request %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	respBody, _ := io.ReadAll(resp.Body)
	var respJson Records
	err = json.Unmarshal(respBody, &respJson)
	if err != nil {
		return "", fmt.Errorf("Failure sendGetRecord : Unmarshal failed %w", err)
	}

	var found_record_id string = ""
	for _, element := range respJson.Records {
		if element.Name == record_name {
			if element.Type == record_type {
				if found_record_id != "" {
					return "", fmt.Errorf("Failure : Found more than one record matching record_name: %s and recrod_type: %s", record_name, record_type)
				}
				found_record_id = element.Id
			}
		}
	}
	if found_record_id != "" {
		return found_record_id, nil
	}
	return "", fmt.Errorf("Failure : Found no records matching record_name: %s and recrod_type: %s", record_name, record_type)
}

func sendPutRecord(token string, zone_id string, record_id string, record_name string, record_ip string, record_type string) (string, error) {
	var ttl uint64 = 86400
	jsonString := `{"value": "` + record_ip + `","ttl": ` + fmt.Sprint(ttl) + `,"type": "` + record_type + `","name": "` + record_name + `","zone_id": "` + zone_id + `"}`
	json := []byte(jsonString)
	body := bytes.NewBuffer(json)

	client := &http.Client{}
	req, _ := http.NewRequest("PUT", "https://dns.hetzner.com/api/v1/records/"+record_id, body)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Auth-API-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failure sendPutRecord : Making Request %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	respBody, _ := io.ReadAll(resp.Body)
	return string(respBody), err

}

func sendGetIp() (string, error) {
	resp, err := http.Get("https://checkip.amazonaws.com")
	if err != nil {
		return "", fmt.Errorf("Failure sendGetIp : Making Reqest %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	respBody, err := io.ReadAll(resp.Body)
	respString := strings.TrimSuffix(string(respBody), "\n")
	if err != nil {
		return "", fmt.Errorf("Failure : Couldn't get ip")
	}
	return respString, nil
}

func isValidRecordType(record_type string) bool {
	switch record_type {
	case "A":
		return true
	case "AAAA":
		return true
	case "NS":
		return true
	case "MX":
		return true
	case "CNAME":
		return true
	case "RP":
		return true
	case "TXT":
		return true
	case "SOA":
		return true
	case "HINFO":
		return true
	case "SRV":
		return true
	case "DANE":
		return true
	case "TLSA":
		return true
	case "DS":
		return true
	case "CAA":
		return true
	default:
		return false
	}
}

func main() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Read Config Failed")
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)

	var auth_token string = config.Token
	var zone_name string = config.Zone_Name
	var record_name string = config.Record_Name
	var record_type string = config.Record_Type
	if !isValidRecordType(record_type) {
		panic(fmt.Errorf("Error : record_type %s is not valid", record_type))
	}

	var record_name_new string = record_name
	if config.Change_Record.Change_Name {
		record_name_new = config.Change_Record.New_Name
	}
	var record_type_new string = record_type
	if config.Change_Record.Change_Type {
		record_type_new = config.Change_Record.New_Type
		if !isValidRecordType(record_type_new) {
			panic(fmt.Errorf("Error : new_type %s is not valid", record_type_new))
		}
	}
	var record_value_new string
	if config.Change_Record.Change_Value {
		if config.Change_Record.Change_Value_To_Wanip {
			record_value_new, err = sendGetIp()
			if err != nil {
				panic(err)
			}
		} else {
			record_value_new = config.Change_Record.New_Value
		}
	}
	fmt.Printf("Loaded Config changing:\n")
	fmt.Printf("In zone %s\n", zone_name)
	fmt.Printf("From Record Name: %s to %s\n", record_name, record_name_new)
	fmt.Printf("From Record Type: %s to %s\n", record_type, record_type_new)
	fmt.Printf("To Record Value: %s\n", record_value_new)

	zone_id, err := sendGetZone(auth_token, zone_name)
	if err != nil {
		panic(err)
	}

	record_id, err := sendGetRecord(auth_token, zone_id, record_name, record_type)
	if err != nil {
		panic(err)
	}

	success, err := sendPutRecord(auth_token, zone_id, record_id, record_name_new, record_value_new, record_type_new)
	if err != nil {
		panic(err)
	}
	fmt.Println(success)

}
