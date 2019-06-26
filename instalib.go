package main

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type API struct {
	VERSION      string
	KeyVersion   string
	KEY          string
	CAPABILITIES string
}

type HttpResponse struct {
	Err       error
	ResStatus int
	Req       *http.Request
	Res       *http.Response
	Body      string
	Headers   http.Header
	Cookies   *cookiejar.Jar
}

func MakeHttpResponse(Response *http.Response, Request *http.Request, jar *cookiejar.Jar, Error error) HttpResponse {

	var res = ""
	var StatusCode = 0
	var Headers http.Header = nil

	if Response != nil {
		var reader io.ReadCloser
		switch Response.Header.Get("Content-Encoding") {
		case "gzip":
			reader, _ = gzip.NewReader(Response.Body)
			defer reader.Close()
		default:
			reader = Response.Body
		}
		body, _ := ioutil.ReadAll(reader)
		res = string(body)

		if Response.Header != nil {
			Headers = Response.Header
		}

		if Response.StatusCode != 0 {
			StatusCode = Response.StatusCode
		}
	}

	return HttpResponse{ResStatus: StatusCode, Res: Response, Req: Request, Body: res, Headers: Headers, Cookies: jar, Err: Error}
}

func createKeyValuePairs(m http.Header) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		_, _ = fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
	}
	return b.String()
}

func HMACSHA256(message string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func GetAPI() API {
	resp, err := http.Get("https://raw.githubusercontent.com/mgp25/Instagram-API/master/src/Constants.php")

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	content := string(body)
	IG_VERSION := regexp.MustCompile("const IG_VERSION = '(.*?)';").FindStringSubmatch(content)[1]
	IG_SIG_KEY := regexp.MustCompile("const IG_SIG_KEY = '(.*?)';").FindStringSubmatch(content)[1]
	SIG_KEY_VERSION := regexp.MustCompile("const SIG_KEY_VERSION = '(.*?)';").FindStringSubmatch(content)[1]
	X_IG_Capabilities := regexp.MustCompile("const X_IG_Capabilities = '(.*?)';").FindStringSubmatch(content)[1]

	_API := API{VERSION: IG_VERSION, KEY: IG_SIG_KEY, KeyVersion: SIG_KEY_VERSION, CAPABILITIES: X_IG_Capabilities}

	return _API
}

func IR(iurl string, signedbody map[string]string, payload string,
	Headers map[string]string, api API, proxy string,
	ptype string, cookie *cookiejar.Jar, usecookies bool) HttpResponse {

	_url := iurl

	if ((!strings.Contains(_url, "https")) || (!strings.Contains(_url, "http"))) && _url[0] != '/' {
		_url = "https://i.instagram.com/api/v1/" + _url
	} else if ((!strings.Contains(_url, "https")) || (!strings.Contains(_url, "http"))) && _url[0] == '/' {
		_url = "https://i.instagram.com/api/v1" + _url
	}

	_api := API{}
	if api == (API{}) {
		_api = GetAPI()
	} else {
		_api = api
	}

	_payload := ""
	if signedbody != nil {
		_data, _ := json.Marshal(signedbody)
		_json := string(_data)
		_signed := fmt.Sprintf("%v.%s", HMACSHA256(_api.KEY, _json), _json)
		_payload = "ig_sig_key_version=" + _api.KeyVersion + "&signed_body=" + _signed
	} else if payload != "" {
		_payload = payload
	}

	var req *http.Request
	if _payload != "" {
		req, _ = http.NewRequest("POST", _url, bytes.NewBuffer([]byte(_payload)))
	} else {
		req, _ = http.NewRequest("GET", _url, nil)
	}

	req.Header.Set("User-Agent", "Instagram "+_api.VERSION+" Android (19/4.4.2; 480dpi; 1080x1920; samsung; SM-N900T; hltetmo; qcom; en_US)")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Cookie2", "$Version=1")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-IG-Connection-Type", "WIFI")
	req.Header.Set("X-IG-Capabilities", _api.CAPABILITIES)
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("X-FB-HTTP-Engine", "Liger")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "Keep-Alive")

	if Headers != nil {
		var keys []string
		for key := range Headers {
			keys = append(keys, key)
		}
		var values []string
		for _, value := range Headers {
			values = append(values, value)
		}

		for i := 0; i < len(keys); i++ {
			req.Header.Set(keys[i], values[i])
		}
	}

	jar := cookie
	transport := http.Transport{}
	if proxy != "" {
		proxyUrl, _ := url.Parse(ptype + "://" + proxy)
		transport.Proxy = http.ProxyURL(proxyUrl) // set proxy proxyType://proxyIp:proxyPort
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //set ssl
	client := &http.Client{}
	if usecookies {
		client = &http.Client{Jar: jar}
	}
	client.Transport = &transport
	resp, err := client.Do(req)
	if err != nil {
		return MakeHttpResponse(resp, req, jar, err)
	}
	defer resp.Body.Close()
	return MakeHttpResponse(resp, req, jar, err)
}

func MakeList(chars []string, l int) []string {
	var list []string
	var clearList []string
	var n = len(chars)
	ml(chars, "", n, l, &list)
	for _, v := range list {
		if v[:1] == "." || v[(len(v)-1):] == "." {
		} else {
			clearList = append(clearList, v)
		}
	}
	return clearList
}

func ml(chars []string, prefix string, n int, l int, list *[]string) {
	var copied []string
	if l == 0 {
		copied = *list
		copied = append(copied, prefix)
		*list = copied
		return
	}
	for i := 0; i < n; i++ {
		newPrefix := prefix + chars[i]
		ml(chars, newPrefix, n, l-1, list)
	}
}

func CreateUsernames(chars []string, length int) []string {
	t := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "_", "."}
	l := 3
	if length != 0 {
		l = length
	}
	if chars != nil {
		t = chars
	}
	return MakeList(t, l)
}
