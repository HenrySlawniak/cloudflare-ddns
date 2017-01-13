package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// Version is our current version
const Version = "1.0.0"

var (
	client                = &http.Client{}
	key                   = flag.String("key", "", "Your cloudflare API key, overrides environment variable CLOUDFLARE_DDNS_KEY")
	email                 = flag.String("email", "", "Your cloudflare API email, overrides environment variable CLOUDFLARE_DDNS_EMAIL")
	domain                = flag.String("domain", "", "The domain to update records on, overrides environment variable CLOUDFLARE_DDNS_DOMAIN")
	subDomain             = flag.String("subdomain", "@", "The subdomain to update records on, overrides environment variable CLOUDFLARE_DDNS_SUBDOMAIN")
	v6                    = flag.Bool("v6", true, "Controls whether or not to set AAAA record")
	v4                    = flag.Bool("v4", true, "Controls whether or not to set A record")
	externalAddressSource = flag.String("external-source", "ifcfg.org", "The external service to use for determining external address, should have v4 and v6 subdomains")
	externalSourceUseSSL  = flag.Bool("external-source-ssl", true, "Whether to use SSL to connect to the external source for the external address")

	finalKey       string
	finalEmail     string
	finalDomain    string
	finalSubdomain string
)

// Zone encodes a cloudflare API Zone
type Zone struct {
	ID   string
	Name string
}

// ZoneList encodes a cloudflare API zone lists
type ZoneList struct {
	Success  bool
	Errors   []Error
	Messages []string
	Result   []Zone
}

// Record encodes a cloudflare API DNS record
type Record struct {
	ID        string
	Type      string
	Name      string
	Content   string
	Locked    bool
	ZoneID    string `json:"zone_id"`
	ZoneNames string `json:"zone_name"`
}

// RecordList encodes a list cloudflare API DNS records
type RecordList struct {
	Success  bool
	Errors   []Error
	Messages []string
	Result   []Record
}

// Error encodes errors from the cloudflare API
type Error struct {
	Code    int
	Message string
}

func main() {
	flag.Parse()
	cLog := console.New()
	cLog.SetTimestampFormat(time.RFC3339)
	log.RegisterHandler(cLog, log.AllLevels...)

	log.Info("Starting cloudflare-ddns v" + Version)

	finalKey = os.Getenv("CLOUDFLARE_DDNS_KEY")
	if *key != "" {
		finalKey = *key
	}

	finalEmail = os.Getenv("CLOUDFLARE_DDNS_EMAIL")
	if *email != "" {
		finalEmail = *email
	}

	finalDomain = os.Getenv("CLOUDFLARE_DDNS_DOMAIN")
	if *domain != "" {
		finalDomain = *domain
	}

	finalSubdomain = os.Getenv("CLOUDFLARE_DDNS_SUBDOMAIN")
	if *subDomain != "" {
		finalSubdomain = *subDomain
	}

	log.Infof("Updating record for %s.%s\n", finalSubdomain, finalDomain)
	log.Infof("\tv4: %t\n", *v4)
	log.Infof("\tv6: %t\n", *v6)
	if *v4 {
		log.Infof("external IPv4 address: %s", GetV4Address())
	}

	if *v6 {
		log.Infof("external IPv6 address: %s", GetV6Address())
	}

	err := UpdateIP()
	if err != nil {
		log.Error(err)
	} else {
		log.Info("Successfully updated DNS records")
	}
}

// UpdateIP will update the DNS records by determining the external IP address
func UpdateIP() error {
	req, _ := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/zones?name="+finalDomain+"&status=active&page=1&per_page=1&order=status&direction=desc&match=all", nil)
	req.Header.Add("X-Auth-Email", finalEmail)
	req.Header.Add("X-Auth-Key", finalKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	zList := ZoneList{}
	json.Unmarshal(body, &zList)

	if !zList.Success {
		return fmt.Errorf("%d: %s", zList.Errors[0].Code, zList.Errors[0].Message)
	}
	id := zList.Result[0].ID

	req, _ = http.NewRequest("GET", "https://api.cloudflare.com/client/v4/zones/"+id+"/dns_records?name="+finalSubdomain+"."+finalDomain+"&page=1&per_page=20&order=type&direction=desc&match=all", nil)
	req.Header.Add("X-Auth-Key", finalKey)
	req.Header.Add("X-Auth-Email", finalEmail)
	req.Header.Add("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	rList := RecordList{}
	json.Unmarshal(body, &rList)

	for _, record := range rList.Result {
		if record.Type == "A" && *v4 {
			err = UpdateRecord(id, record.ID, record.Type, record.Name, GetV4Address())
			if err != nil {
				return err
			}
		}

		if record.Type == "AAAA" && *v6 {
			err = UpdateRecord(id, record.ID, record.Type, record.Name, GetV6Address())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateRecord updates a cloudflare DNS record, and returns any error
func UpdateRecord(zoneID, recordID, recordType, recordName, recordContent string) error {
	data := []byte(`{"id": "` + recordID + `", "content": "` + recordContent + `", "type": "` + recordType + `", "name": "` + recordName + `", "ttl": 120}`)
	req, _ := http.NewRequest("PUT", "https://api.cloudflare.com/client/v4/zones/"+zoneID+"/dns_records/"+recordID, bytes.NewBuffer(data))
	req.Header.Add("X-Auth-Key", finalKey)
	req.Header.Add("X-Auth-Email", finalEmail)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	rList := RecordList{}
	json.Unmarshal(body, &rList)

	if !rList.Success {
		return fmt.Errorf("%d: %s", rList.Errors[0].Code, rList.Errors[0].Message)
	}

	return nil
}

// GetV4Address uses the configured external source to get your external IPv4 address
func GetV4Address() string {
	url := "v4." + *externalAddressSource
	if *externalSourceUseSSL {
		url = "https://" + url
	} else {
		url = "http://" + url
	}
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	address, _ := ioutil.ReadAll(resp.Body)

	return string(bytes.TrimSpace(address))
}

// GetV6Address uses the configured external source to get your external IPv6 address
func GetV6Address() string {
	url := "v6." + *externalAddressSource
	if *externalSourceUseSSL {
		url = "https://" + url
	} else {
		url = "http://" + url
	}
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	address, _ := ioutil.ReadAll(resp.Body)

	return string(bytes.TrimSpace(address))
}
