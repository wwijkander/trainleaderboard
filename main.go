package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	trvUrl = "https://api.trafikinfo.trafikverket.se/v2/data.json"
	debug  = os.Getenv("TT_DEBUG")
	trvKey = os.Getenv("TT_TRV_KEY")

	client = &http.Client{}
)

type XMLRequest struct {
	XMLName xml.Name        `xml:"request"`
	Login   XMLRequestLogin `xml:"login"`
	Query   XMLRequestQuery `xml:"query"`
}

type XMLRequestLogin struct {
	AuthenticationKey string `xml:"authenticationkey,attr"`
}

type XMLRequestQuery struct {
	Namespace     string    `xml:"namespace,attr"`
	ObjectType    string    `xml:"objecttype,attr"`
	SchemaVersion string    `xml:"schemaversion,attr"`
	Limit         int       `xml:"limit,attr"`
	OrderBy       string    `xml:"orderby,attr"`
	FilterEq      XMLFilter `xml:"filter>eq"`
	FilterGt      XMLFilter `xml:"filter>gt"`
	Include       []string  `xml:"include"`
}

type XMLFilter struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type TrainPositionReply struct {
	Response struct {
		Result []struct {
			TrainPosition []struct {
				Train struct {
					OperationalTrainNumber        string    `json:"OperationalTrainNumber"`
					OperationalTrainDepartureDate time.Time `json:"OperationalTrainDepartureDate"`
					JourneyPlanNumber             string    `json:"JourneyPlanNumber"`
					JourneyPlanDepartureDate      time.Time `json:"JourneyPlanDepartureDate"`
					AdvertisedTrainNumber         string    `json:"AdvertisedTrainNumber"`
				} `json:"Train"`
				Position struct {
					Sweref99Tm string `json:"SWEREF99TM"`
					Wgs84      string `json:"WGS84"`
				} `json:"Position"`
				TimeStamp time.Time `json:"TimeStamp"`
				Status    struct {
					Active bool `json:"Active"`
				} `json:"Status"`
				Bearing      int       `json:"Bearing"`
				Speed        int       `json:"Speed"`
				ModifiedTime time.Time `json:"ModifiedTime"`
				Deleted      bool      `json:"Deleted"`
			} `json:"TrainPosition"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}

type Train struct {
	lat       float64
	long      float64
	number    int
	bearing   int
	speed     int
	timestamp string
}

func xmlMarshal(request XMLRequest) []byte {
	xmlstring, err := xml.MarshalIndent(request, "", "    ")
	if err != nil {
		panic(err)
	}
	xmlbytes := []byte(xml.Header + string(xmlstring))
	return xmlbytes
}

func doTrvRequest(xmlbytes []byte) {
	req, err := http.NewRequest("POST", trvUrl, bytes.NewBuffer(xmlbytes))
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "traintiles/v0.0-prerelease")
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	decoded := json.NewDecoder(resp.Body)

	var reply TrainPositionReply

	err = decoded.Decode(&reply)
	if err != nil {
		panic(err)
	}
	for _, result := range reply.Response.Result {
		for k, train := range result.TrainPosition {
			fmt.Println(strconv.Itoa(k) + ": Tåg nummer " + train.Train.AdvertisedTrainNumber + " kör i " + strconv.Itoa(train.Speed) + "km/h, är vid " + train.Position.Wgs84)
		}
	}

}

func main() {

	if len(trvKey) <= 0 {
		log.Fatalln("no Trafikverket key set!")
	}
	v := XMLRequest{
		Login: XMLRequestLogin{AuthenticationKey: trvKey},
		Query: XMLRequestQuery{
			Namespace:     "järnväg.trafikinfo",
			ObjectType:    "TrainPosition",
			SchemaVersion: "1.0",
			OrderBy:       "Speed desc",
			Limit:         25,
			FilterEq: XMLFilter{
				Name:  "Status.Active",
				Value: "true",
			},
			FilterGt: XMLFilter{
				Name:  "ModifiedTime",
				Value: "$dateadd(-0.00:15:00)",
			},
			Include: []string{"Train.AdvertisedTrainNumber", "Position.WGS84", "Bearing", "Speed", "TimeStamp"},
		},
	}
	xmlstring := xmlMarshal(v)
	//fmt.Printf("%s\n", xmlstring)
	doTrvRequest(xmlstring)

}
