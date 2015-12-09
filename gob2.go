package gob2

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
)

const (
	baseURL = "https://api.backblaze.com/b2api/v1/"
)

var (
	authorizationToken string
	accountID          string
	uploadURL          string
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
		return "", errors.New("Invalid status code: " + string(resp.StatusCode)) // TODO: get more info from response
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bucketID = getFromJSON(body, "bucketId")

	return
}

// DeleteBucket removes a bucket by ID.
func DeleteBucket(bucketID string) (err error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", baseURL+"b2_delete_bucket", nil)
	req.Header.Add("Authorization", authorizationToken)
	req.PostForm.Add("accountId", accountID)
	req.PostForm.Add("bucketId", bucketID)

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		err = errors.New("Invalid status code: " + string(resp.StatusCode)) // TODO: get more info from response
	}

	return
}

// DeleteFileVersion deletes a version of a file
func DeleteFileVersion(fileName string, fileID string) (err error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", baseURL+"b2_delete_file_version", nil)
	req.Header.Add("Authorization", authorizationToken)
	req.PostForm.Add("fileName", fileName)
	req.PostForm.Add("fileId", fileID)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		err = errors.New("Invalid status code: " + string(resp.StatusCode)) // TODO: Get more info from response
	}

	return
}

// DownloadFileByID can download a file given a fileID, plus the uploadURL as returned by GetUploadURL
func DownloadFileByID(fileID string, uploadURL string, outDir string) (err error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", uploadURL+"b2_download_file_by_id", nil)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		err = errors.New("Invalid status code: " + string(resp.StatusCode)) // TODO: Get more info from response
	}

	fileName := resp.Header.Get("X-Bz-File-Name")
	remoteSHA := resp.Header.Get("X-Bz-Content-Sha1")

	defer resp.Body.Close()
	out, err := os.Create(outDir + fileName) // Check slashing
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(out, resp.Body)

	localSHA, err := getSHA1(outDir + fileName)
	if err != nil {
		return err
	}
	if localSHA != remoteSHA {
		err = errors.New("The sha1sum of the remote file doesn't match what was downloaded. Please retry.")
	}

	return
}

// GetUploadURL gets the URL which should be used when uploading a file
func GetUploadURL(bucketID string) (err error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", baseURL+"b2_get_upload_url", nil)
	req.Header.Add("Authorization", authorizationToken)
	req.PostForm.Add("bucketId", bucketID)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		err = errors.New("Error: no upload URL returned.")
	}
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

func getSHA1(filepath string) (string, error) {
	const chunkSize = 8192

	file, err := os.Open(filepath)

	if err != nil {
		return "", err
	}

	defer file.Close()

	// calculate the file size
	info, _ := file.Stat()

	filesize := info.Size()

	blocks := uint64(math.Ceil(float64(filesize) / float64(chunkSize)))

	hash := sha1.New()

	for i := uint64(0); i < blocks; i++ {
		blocksize := int(math.Min(chunkSize, float64(filesize-int64(i*chunkSize))))
		buf := make([]byte, blocksize)

		file.Read(buf)
		io.WriteString(hash, string(buf)) // append into the hash
	}

	return string(hash.Sum(nil)), nil
}
