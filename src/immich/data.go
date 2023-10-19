package immich

import (
	"encoding/json"
	"fmt"
	"immich-exp/src/models"
	"io/ioutil"

	"net/http"
	"sync"

	prom "immich-exp/src/prometheus"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

var wg sync.WaitGroup

var unmarshalError = "Can not unmarshal JSON"

func Allrequests(r *prometheus.Registry) {

	wg.Add(1)
	go ServerVersion(r)
	wg.Add(1)
	go Analyze(r)
	wg.Wait()
}

func Analyze(r *prometheus.Registry) {
	defer wg.Done()

	allusers := make(chan func() (*models.StructAllUsers, error))
	serverinfo := make(chan func() (*models.StructServerInfo, error))

	wg.Add(1)
	go GetAllUsers(allusers)
	res1, err := (<-allusers)()
	wg.Add(1)
	go ServerInfo(serverinfo)

	res2, err2 := (<-serverinfo)()

	if err != nil && err2 != nil {
	} else {
		prom.SendBackMessagePreference(res2, res1, r)
	}
	close(serverinfo)
	close(allusers)
}

func GetAllUsers(c chan func() (*models.StructAllUsers, error)) {
	defer wg.Done()
	resp, err := Apirequest("/api/user?isAll=true", "GET")
	if err == nil {
		if models.GetPromptError() == true {
			models.SetPromptError(false)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		} else {

			result := new(models.StructAllUsers)
			if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
				log.Error(unmarshalError)
			}

			c <- (func() (*models.StructAllUsers, error) { return result, nil })
		}
	}
}

func ServerVersion(r *prometheus.Registry) {
	defer wg.Done()
	resp, err := Apirequest("/api/server-info/version", "GET")
	if err == nil {
		if models.GetPromptError() == true {
			models.SetPromptError(false)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		} else {

			var result models.StructServerVersion
			if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
				log.Error(unmarshalError)
			}

			prom.SendBackMessageserverVersion(&result, r)
		}
	}
}

func ServerInfo(c chan func() (*models.StructServerInfo, error)) {
	defer wg.Done()
	resp, err := Apirequest("/api/server-info/stats", "GET")
	if err == nil {

		if models.GetPromptError() == true {
			models.SetPromptError(false)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		} else {

			result := new(models.StructServerInfo)
			if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
				log.Println(unmarshalError)
			}
			c <- (func() (*models.StructServerInfo, error) { return result, nil })

		}
	}
}

func Apirequest(uri string, method string) (*http.Response, error) {

	req, err := http.NewRequest(method, models.Getbaseurl()+uri, nil)
	if err != nil {
		log.Fatal("Error with url")
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("x-api-key", models.GetApiKey())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		err := fmt.Errorf("Can't connect to server")
		if models.GetPromptError() == false {
			log.Error(err.Error())
			models.SetPromptError(true)
		}

		return resp, err

	}
	switch resp.StatusCode {
	case http.StatusOK:
		if models.GetPromptError() {
			models.SetPromptError(false)
		}
		return resp, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		err := fmt.Errorf("%d", resp.StatusCode)

		log.Fatal("Api key unauthorized")

		return resp, err
	default:
		err := fmt.Errorf("%d", resp.StatusCode)
		if !models.GetPromptError() {
			models.SetPromptError(true)
			log.Debug("Error code ", resp.StatusCode)
		}
		return resp, err
	}

}
