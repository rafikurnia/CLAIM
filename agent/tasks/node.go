package tasks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type nodeInfo struct {
	Status        string  `json:"status"`
	Message       string  `json:"message,omitempty"`
	Continent     string  `json:"continent"`
	ContinentCode string  `json:"continentCode"`
	Country       string  `json:"country"`
	CountryCode   string  `json:"countryCode"`
	Region        string  `json:"region"`
	RegionName    string  `json:"regionName"`
	City          string  `json:"city"`
	District      string  `json:"district"`
	Zip           string  `json:"zip"`
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Timezone      string  `json:"timezone"`
	Offset        int     `json:"offset"`
	Currency      string  `json:"currency"`
	ISP           string  `json:"isp"`
	Org           string  `json:"org"`
	AS            string  `json:"as"`
	ASName        string  `json:"asname"`
	Reverse       string  `json:"reverse"`
	Mobile        bool    `json:"mobile"`
	Proxy         bool    `json:"proxy"`
	Hosting       bool    `json:"hosting"`
	Query         string  `json:"query"`
}

func GetNodeInfo() (*nodeInfo, error) {
	client := http.Client{
		Timeout: 200 * time.Millisecond,
	}
	req, err := client.Get(
		"http://ip-api.com/json?fields=" +
			"status," +
			"message," +
			"continent," +
			"continentCode," +
			"country," +
			"countryCode," +
			"region," +
			"regionName," +
			"city," +
			"district," +
			"zip," +
			"lat," +
			"lon," +
			"timezone," +
			"offset," +
			"currency," +
			"isp," +
			"org," +
			"as," +
			"asname," +
			"reverse," +
			"mobile," +
			"proxy," +
			"hosting," +
			"query",
	)

	if err != nil {
		return nil, fmt.Errorf("http.Get -> %w", err)
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll -> %w", err)
	}

	nodeInfo := &nodeInfo{}
	json.Unmarshal(body, nodeInfo)

	return nodeInfo, nil
}
