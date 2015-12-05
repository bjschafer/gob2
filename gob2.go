package gob2

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

const (
	baseURL = "https://api.backblaze.com/b2api/v1/"
)

var (
	authorizationToken string
	accountID          string
)

// AuthorizeAccount authorizes an account. First thing needed to be done
func AuthorizeAccount(accID string, appKey string) (err error) {
	accountID = accID // am i just dumb? or lazy.
	authString := base64.StdEncoding.EncodeToString([]byte(accountID + ":" + appKey))
	client := &http.Client{}

	req, err := http.NewRequest("GET", baseURL+"b2_authorize_account", nil)
	req.Header.Add("Authorization", "Basic "+authString)
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	authorizationToken = getFromJSON(body, "authorizationToken")
	return
}

// CreateBucket creates a new bucket with the given name and type
// either allPublic or allPrivate
func CreateBucket(bucketName string, bucketType string) (bucketID string, err error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", baseURL+"b2_create_bucket", nil)
	req.Header.Add("Authorization", authorizationToken)
	req.PostForm.Add("accountId", accountID)
	req.PostForm.Add("bucketName", bucketName)
	req.PostForm.Add("bucketType", bucketType)

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("Invalid status code: " + string(resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bucketID = getFromJSON(body, "bucketId")

	return
}

func getFromJSON(inJSON []byte, key string) (val string) {
	var data interface{}
	err := json.Unmarshal(inJSON, data)
	if err != nil {
		panic(err)
	}

	d := data.(map[string]string)

	val = d[key]
	return
}
