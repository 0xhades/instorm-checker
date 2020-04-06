package main

import (
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

func login(us string, ps string) HttpResponse {
	url := "https://i.instagram.com/api/v1/accounts/login/"

	jar, _ := cookiejar.New(nil)

	u, _ := uuid.NewUUID()
	guid := u.String()

	post := make(map[string]string)
	post["phone_id"] = guid
	post["_csrftoken"] = "missing"
	post["username"] = us
	post["password"] = ps
	post["device_id"] = guid
	post["guid"] = guid
	post["login_attempt_count"] = "0"

	return IR(url, post, "", nil, InstaAPI, "", "", jar, true)
}

func GetProfile(jar cookiejar.Jar, api API) (map[string]string, HttpResponse) {
	res := IR("accounts/current_user/?edit=true", nil, "", nil, api, "", "", &jar, true)
	var profile = make(map[string]string)

	var username = ""
	_username := regexp.MustCompile("\"username\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _username != nil {
		username = _username[1]
	}
	var biography = ""
	_biography := regexp.MustCompile("\"biography\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _biography != nil {
		biography = _biography[1]
	}

	var fullName = ""
	_fullName := regexp.MustCompile("\"full_name\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _fullName != nil {
		fullName = _fullName[1]
	}

	var phoneNumber = ""
	_phoneNumber := regexp.MustCompile("\"phone_number\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _phoneNumber != nil {
		phoneNumber = _phoneNumber[1]
	}

	var email = ""
	_email := regexp.MustCompile("\"email\": \"(.*?)\"").FindStringSubmatch(res.Body)
	if _email != nil {
		email = _email[1]
	}
	var gender = ""
	_gender := regexp.MustCompile("\"gender\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _gender != nil {
		gender = _gender[1]
	}

	var externalUrl = ""
	_externalUrl := regexp.MustCompile("\"external_url\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _externalUrl != nil {
		externalUrl = _externalUrl[1]
	}

	var isVerified = ""
	_isVerified := regexp.MustCompile("\"is_verified\": \"(.*?)\",").FindStringSubmatch(res.Body)
	if _isVerified != nil {
		isVerified = _isVerified[1]
	}

	profile["username"] = username
	profile["biography"] = biography
	profile["full_name"] = fullName
	profile["phone_number"] = phoneNumber
	profile["email"] = email
	profile["gender"] = gender
	profile["external_url"] = externalUrl
	profile["is_verified"] = isVerified

	return profile, res
}

func ssliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func edit(jar *cookiejar.Jar, res HttpResponse /*Login() Cookies*/, username string, email string, phone_number string, external_url string, biography string, gender string, full_name string) HttpResponse {

	var errs = 0
	for {

		cookie := createKeyValuePairs(res.Res.Header)
		csrftoken := regexp.MustCompile("csrftoken=(.*?);").FindStringSubmatch(cookie)[1]
		pk := regexp.MustCompile("ds_user_id=(.*?);").FindStringSubmatch(cookie)[1]
		profile, _ := GetProfile(*jar, InstaAPI)

		var _username string
		var _email string
		var _phone_number string
		var _external_url string
		var _biography string
		var _gender string
		var _full_name string

		if username != "" {
			_username = username
		} else {
			_username = profile["username"]
		}
		if email != "" {
			_email = email
		} else {
			_email = profile["email"]
		}
		if phone_number != "" {
			_phone_number = phone_number
		} else {
			_phone_number = profile["phone_number"]
		}
		if external_url != "" {
			_external_url = external_url
		} else {
			_external_url = profile["external_url"]
		}
		if biography != "" {
			_biography = biography
		} else {
			_biography = profile["biography"]
		}
		if gender != "" {
			_gender = gender
		} else {
			_gender = profile["gender"]
		}
		if full_name != "" {
			_full_name = full_name
		} else {
			_full_name = profile["full_name"]
		}

		_url := "https://i.instagram.com/api/v1/accounts/edit_profile/"

		u, _ := uuid.NewUUID()
		guid := u.String()

		postData := make(map[string]string)
		postData["external_url"] = _external_url
		postData["_uid"] = pk
		postData["_uuid"] = guid
		postData["biography"] = _biography
		postData["_csrftoken"] = csrftoken
		postData["username"] = _username
		if strings.Contains(_email, "+") {
			_email = strings.Replace(_email, "+", "", -1)
		}
		postData["email"] = _email
		postData["full_name"] = _full_name
		postData["phone_number"] = _phone_number
		if _gender != "" {
			postData["gender"] = _gender //1 = male, 2 = female
		} else {
			postData["gender"] = "1" //1 = male, 2 = female
		}

		res := IR(_url, postData, "", nil, InstaAPI, "", "", jar, true)
		if res.Err == nil {
			return res
		} else {
			if errs > 50 {
				panic(res.Err)
			}
			errs++
		}
	}
}

func CheckWebInstagram(us string, sessionid string, proxy string /*example: ( IpProxy:IpPort )*/, ptype string) bool {
	var hosterr = 0
	var errs = 0
	for {
		var req *http.Request
		req, _ = http.NewRequest("GET", "https://www.instagram.com/"+us+"/?__a=1", nil)
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36")

		transport := http.Transport{}
		if proxy != "" {
			proxyUrl, _ := url.Parse(ptype + "://" + proxy)
			transport.Proxy = http.ProxyURL(proxyUrl) // set proxy proxyType://proxyIp:proxyPort
		}

		jar, _ := cookiejar.New(nil)
		var cookies []*http.Cookie
		var cookie = &http.Cookie{}
		cookie = &http.Cookie{
			Name:   "sessionid",
			Value:  sessionid,
			Path:   "/",
			Domain: "instagram.com",
		}
		cookies = append(cookies, cookie)
		u, _ := url.Parse("https://i.instagram.com/api/v1/accounts/login/")
		jar.SetCookies(u, cookies)

		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //set ssl
		client := &http.Client{Jar: jar}
		client.Transport = &transport
		resp, err := client.Do(req)

		var reader io.ReadCloser
		var response = ""

		if resp != nil {
			switch resp.Header.Get("Content-Encoding") {
			case "gzip":
				reader, _ = gzip.NewReader(resp.Body)
				defer reader.Close()
			default:
				reader = resp.Body
			}
			body, _ := ioutil.ReadAll(reader)
			response = string(body)
		}

		if err != nil || response == "" || resp == nil {
			if response != "" {
				fmt.Println(response)
			}
			if errs == 10 {
				if resp != nil {
					fmt.Println(req)
					fmt.Println(resp)
					fmt.Println(resp.Header)
					fmt.Println(resp.StatusCode)
				}
				if err != nil {
					fmt.Println(us)
					panic(err)
				} else {
					fmt.Println(us)
					os.Exit(0)
				}
			} else {
				if strings.Contains(err.Error(), "no such host") { //block
					if hosterr <= 5 {
						hosterr++
						time.Sleep(10 * time.Second)
					} else {
						if hosterr == 10 {
							panic(err)
						}
						if len(CheckCookiesList) == 0 {
							CookieIndex = 0
						} else {
							if CookieIndex >= len(CheckCookiesList) {
								CookieIndex = 0
							} else {
								CookieIndex++
							}
						}
						CheckCookies = CheckCookiesList[CookieIndex]
					}
				} else {
					errs++
				}
				continue
			}
		}
		defer resp.Body.Close()

		/*if !(len(response) <= 5) {
			fmt.Println(response[:20])
		}*/

		if (!strings.Contains(response, "logging_page_id") && strings.Contains(response, "Page Not Found")) || (!strings.Contains(response, "logging_page_id") && strings.Contains(response, "{") && response == "{}") {
			if CheckUserName(us) {
				return true
			}
		}
		return false
	}
}

func CheckUserName(us string) bool {
	var errs = 0
	for {

		_url := "accounts/create/"

		postData := make(map[string]string)
		u, _ := uuid.NewUUID()
		_guid := u.String()

		postData["phone_id"] = _guid
		postData["_csrftoken"] = "missing"
		postData["username"] = us
		postData["password"] = "hello"
		postData["email"] = "what are want"
		postData["device_id"] = _guid
		postData["guid"] = _guid
		res := IR(_url, postData, "", nil, InstaAPI, "", "", nil, false)
		if res.Err == nil || res.Body == "" {
			if !strings.Contains(res.Body, "username") &&
				res.Body != "" && !strings.Contains(res.Body, "requests") &&
				!strings.Contains(res.Body, "request") &&
				!strings.Contains(res.Body, "please wait") &&
				!strings.Contains(res.Body, "wait") {
				return true
			}
			return false
		} else {
			if errs > 10 {
				fmt.Println(us)
				panic(res.Err)
			}
			errs++
		}
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		_, _ = fmt.Fprintln(w, line)
	}
	return w.Flush()
}

var dir, _ = os.Getwd()
var list []string
var counter = 0
var TPM int
var uslog string
var pslog string
var auto = false
var shutA = false
var cookies *cookiejar.Jar
var LogRes HttpResponse
var Available []string
var Taken []string
var repeat = false
var InstaAPI = GetAPI()
var CheckCookies string
var CheckCookiesList []string
var CookieIndex = 0

func Process(us string) {
	if counter >= len(list) {
		return
	}
	// the error is pure fmt.printf() without varuible
	input := "\r\033[33mProgress:\033[0m \033[34m%d\033[0m/\033[34m%d\033[0m \033[33mAvailable:\033[0m \033[32m%d\033[0m \033[33mTaken:\033[0m \033[31m%d\033[0m"
	output := fmt.Sprintf(input, counter, len(list), len(Available), len(Taken))
	print(output)
	counter++
	if CheckWebInstagram(us, CheckCookies, "", "") {
		if !ssliceContains(Available, us) && us != "" {
			Available = append(Available, us)
		}
		dir += "/available.txt"
		_ = writeLines(Available, dir)
		if auto {
			edit(cookies, LogRes, Available[len(Available)-1], "", "", "", "", "", "")
			if shutA {
				fmt.Println()
				os.Exit(0)
			}
		}
	} else {
		Taken = append(Taken, us)
	}
}

func Start() {
	fmt.Printf("\033[33mProgress:\033[0m \033[34m%d\033[0m/\033[34m%d\033[0m \033[33mAvailable:\033[0m \033[32m%d\033[0m \033[33mTaken:\033[0m \033[31m%d\033[0m", 0, len(list), len(Available), len(Taken))
	Attempts := (len(list) / TPM) + (len(list) % TPM)
	for i := 0; i < Attempts; i++ {
		/*
			if counter >= len(list) {
				if repeat {
					counter = 0
				}
				return
			}
		*/
		wg := sync.WaitGroup{}
		for z := 0; z < TPM; z++ {
			if counter >= len(list) {
				if repeat {
					counter = 0
				}
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				Process(list[counter])
			}()
		}
		wg.Wait()
		if (len(list) - counter) < TPM {
			TPM = len(list) - counter
		}
	}
}

func main() {
	print("\033[H\033[2J")
	R := color.New(color.FgRed, color.Bold)
	G := color.New(color.FgGreen)
	_, _ = R.Println("    _             __")
	_, _ = R.Println("   (_)____  _____/ /_____  _________ ___")
	_, _ = R.Println("  / // __ \\/ ___/ __/ __ \\/ ___/ __ `__ \\")
	_, _ = R.Println(" / // / / (__  ) /_/ /_/ / /  / / / / / /")
	_, _ = R.Println("/_//_/ /_/____/\\__/\\____/_/  /_/ /_/ /_/ ")
	_, _ = R.Println("")
	color.Blue("By BlackHole, inst: @fenllz")
	fmt.Println()
	_, _ = G.Print("Enter a number of threads [best 25] (it depends on your computer): ")
	_, _ = fmt.Scanln(&TPM)
	_, _ = G.Println("Enter the path of the clients (instagram accounts) list (at least 2 account like aaa:123321): ")
	// to bypass the blocking
	var _path string
	_, _ = fmt.Scanln(&_path)
	var accountslist []string
	accountslist, _ = readLines(_path)
	_, _ = G.Println("Get the cookies of the accounts, please wait ...")

	for i := 0; i < len(accountslist); i++ {
		res := login(strings.Split(accountslist[i], ":")[0], strings.Split(accountslist[i], ":")[1])
		if strings.Contains(res.Body, "logged_in_user") {
			_url, _ := url.Parse("https://i.instagram.com/api/v1/accounts/login/")
			var cokkies = res.Cookies.Cookies(_url)
			var sessionid string

			for i := 0; i < len(cokkies); i++ {
				if strings.Contains(strings.ToLower(cokkies[i].Name), "session") && strings.Contains(strings.ToLower(cokkies[i].Name), "id") {
					sessionid = cokkies[i].Value
				}
			}

			CheckCookiesList = append(CheckCookiesList, sessionid)
		}
	}

	CheckCookies = CheckCookiesList[0]

	_, _ = G.Print("Do you want to create new list(n), or use excited one(e) ? [n/e]: ")
	var choice string
	_, _ = fmt.Scanln(&choice)
	if choice != "n" {
		_, _ = G.Println("Enter the path of the list") // the same folder which the list is in, make available.txt.
		_, _ = G.Println("(win: C:\\Users\\something.[txt/list] | [mac/linux]: /Users/something.[txt/list]): ")
		var path string
		_, _ = fmt.Scanln(&path)
		list, _ = readLines(path)
		_, _ = R.Print("The list has: ")
		fmt.Println(len(list))
	} else {
		_, _ = G.Print("You want a list of [n] characters usernames: [(int)n = ?]: ")
		var l int
		_, _ = fmt.Scanln(&l)
		list = CreateUsernames(nil, l)
		dir += "/list.txt"
		_ = writeLines(list, dir)
		_, _ = R.Print("The list has: ")
		fmt.Println(len(list) + 1)
		_, _ = R.Print("Saved path: ")
		fmt.Println(dir)
	}
	_, _ = G.Print("Do want automatically sign the last available username ? [y/n]: ")
	_, _ = fmt.Scanln(&choice)
	if choice == "y" {
		_, _ = R.Println("be aware, if the account's email has a plus sign (+) in it, the plus sign will be removed!")
		time.Sleep(time.Second * 3)
		_, _ = G.Print("Enter an username: ") // I have an idea for multiple accounts to take the available usernames.
		_, _ = fmt.Scanln(&uslog)
		_, _ = G.Print("Enter a password: ") // I have an idea for multiple accounts to take the available usernames.
		_, _ = fmt.Scanln(&pslog)
		res := login(uslog, pslog)
		if strings.Contains(res.Body, "logged_in_user") {
			auto = true
			cookies = res.Cookies
			LogRes = res
			_, _ = G.Print("Do want shutdown after the first available username (after signed it) ? [y/n]: ")
			_, _ = fmt.Scanln(&choice)
			if choice != "y" {
				shutA = true
			}
		} else {
			_, _ = G.Print("There error with login into the account, run the script again.")
			os.Exit(1)
		}
	}
	_, _ = G.Print("Do want repeat the checker after it completed ? [y/n]: ")
	_, _ = fmt.Scanln(&choice)
	if choice != "y" {
		repeat = true
	}
	print("\033[H\033[2J")
	_, _ = R.Println("    _             __")
	_, _ = R.Println("   (_)____  _____/ /_____  _________ ___")
	_, _ = R.Println("  / // __ \\/ ___/ __/ __ \\/ ___/ __ `__ \\")
	_, _ = R.Println(" / // / / (__  ) /_/ /_/ / /  / / / / / /")
	_, _ = R.Println("/_//_/ /_/____/\\__/\\____/_/  /_/ /_/ /_/ ")
	_, _ = R.Println("")
	color.Blue("By BlackHole, inst: @wwvq")
	fmt.Println()
	Start()
	fmt.Printf("\r\033[33mProgress:\033[0m \033[34m%d\033[0m/\033[34m%d\033[0m \033[33mAvailable:\033[0m \033[32m%d\033[0m \033[33mTaken:\033[0m \033[31m%d\033[0m", len(Available)+len(Taken), len(list), len(Available), len(Taken))
	fmt.Println()
	if len(Available) != 0 {
		dir += "/available.txt"
		_ = writeLines(Available, dir)
		_, _ = R.Print("The Available Usernames list has: ")
		fmt.Println(len(Available))
		_, _ = R.Print("Saved path: ")
		fmt.Println(dir)
		fmt.Println()
	}
}
