package cowtransfer

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"time"
	"transfer/apis"
)

func (b cowTransfer) blockPut(postURL string, buf []byte, token string) (string, error) {
	data := new(bytes.Buffer)
	data.Write(buf)
	body, err := newRequest(postURL, data, requestConfig{
		debug:  apis.DebugMode,
		action: "PUT",

		//retry:    0,
		timeout:  time.Duration(b.Config.interval) * time.Second,
		modifier: addToken(token),
	})
	if err != nil {
		if apis.DebugMode {
			log.Printf("block upload failed (retrying)")
		}
		//if retry > 3 {
		return "", err
		//}
		//return b.blockPut(postURL, buf, token, retry+1)
	}
	var rBody uploadResponse
	if err := json.Unmarshal(body, &rBody); err != nil {
		if apis.DebugMode {
			log.Printf("resp unmarshal failed (retrying)")
		}
		//if retry > 3 {
		return "", err
		//}
		//return b.blockPut(postURL, buf, token, retry+1)
	}
	if b.Config.hashCheck {
		if hashBlock(buf) != rBody.MD5 {
			if apis.DebugMode {
				log.Printf("block hashcheck failed (retrying)")
			}
			//if retry > 3 {
			return "", err
			//}
			//return b.blockPut(postURL, buf, token, retry+1)
		}
	}
	if rBody.Error != "" {
		return "", fmt.Errorf(rBody.Error)
	}
	return rBody.Etag, nil
}

func hashBlock(buf []byte) string {
	return fmt.Sprintf("%x", md5.Sum(buf))
}

func newRequest(link string, postBody io.Reader, config requestConfig) ([]byte, error) {
	if config.debug {
		//if config.retry != 0 {
		//	log.Printf("retrying: %v", config.retry)
		//}
		log.Printf("endpoint: %s", link)
	}
	client := http.Client{}
	if config.timeout != 0 {
		client = http.Client{Timeout: config.timeout}
	}
	req, err := http.NewRequest(config.action, link, postBody)
	if err != nil {
		if config.debug {
			log.Printf("build requests error: %v", err)
		}
		//if config.retry > 3 {
		return nil, err
		//}
		//return newPostRequest(link, postBody, config)
	}
	config.modifier(req)
	resp, err := client.Do(req)
	if err != nil {
		if config.debug {
			log.Printf("do requests error: %v", err)
		}
		//if config.retry > 20 {
		return nil, err
		//}
		//config.retry++
		//return newPostRequest(link, postBody, config)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if config.debug {
			log.Printf("read response error: %v", err)
		}
		//if config.retry > 20 {
		return nil, err
		//}
		//config.retry++
		//return newPostRequest(link, postBody, config)
	}
	_ = resp.Body.Close()
	if config.debug {
		if len(body) < 1024 {
			log.Printf("returns: %v", string(body))
		}
	}
	return body, nil
}

func (b cowTransfer) newMultipartRequest(url string, params map[string]string, config requestConfig) ([]byte, error) {
	if config.debug {
		//log.Printf("retrying: %v", config.retry)
		log.Printf("endpoint: %s", url)
	}
	client := http.Client{}
	if config.timeout != 0 {
		client = http.Client{Timeout: config.timeout}
	}
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	_ = writer.Close()
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		if config.debug {
			log.Printf("build requests error: %v", err)
		}
		//if config.retry > 3 {
		return nil, err
		//}
		//config.retry++
		//time.Sleep(1)
		//return b.newMultipartRequest(url, params, config)
	}

	req.Header.Set("content-type", fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary()))
	config.modifier(req)
	if config.debug {
		log.Printf("header: %v", req.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		if config.debug {
			log.Printf("do requests error: %v", err)
		}
		//if config.retry > 3 {
		return nil, err
		//}
		//config.retry++
		//time.Sleep(1)
		//return b.newMultipartRequest(url, params, config)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if config.debug {
			log.Printf("read response returns: %v", err)
		}
		//if config.retry > 3 {
		return nil, err
		//}
		//config.retry++
		//time.Sleep(1)
		//return b.newMultipartRequest(url, params, config)
	}
	_ = resp.Body.Close()
	if config.debug {
		log.Printf("returns: %v", string(body))
	}

	return body, nil
}

func addToken(upToken string) func(req *http.Request) {
	return func(req *http.Request) {
		addHeaders(req)
		req.Header.Set("Authorization", "UpToken "+upToken)
	}
}

func (b cowTransfer) addTk(req *http.Request) {
	ck := b.Config.token
	if b.Config.authCode != "" {
		ck = fmt.Sprintf("%s; cow-auth-token=%s", b.Config.token, b.Config.authCode)
	}

	req.Header.Set("cookie", ck)
	req.Header.Set("authorization", b.Config.authCode)
}

func addHeaders(req *http.Request) {
	req.Header.Set("Referer", "https://cowtransfer.com/")
	req.Header.Set("User-Agent", "Chrome/80.0.3987.149 Transfer")
	req.Header.Set("Origin", "https://cowtransfer.com/")
	req.Header.Set("Cookie", fmt.Sprintf("%scf-cs-k-20181214=%d;", req.Header.Get("Cookie"), time.Now().UnixNano()))
}
