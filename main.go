package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/gookit/color"
)

var mu sync.RWMutex

// TISTORY APP_ID
const secretKey = "c7a92e9bd27df7b405ea3678e03eb460981968cada5a658446c98e955ebd1711bcfe26e9"
const clientID = "c7a92e9bd27df7b405ea3678e03eb460"

// redirectURL은 Tistroy App CallURL로 설정이 되어 있어야지 작동합니다.
const port = "8080"
const redirectURL = "http://localhost:" + port + "/"

func main() {
	var token string

	// Token를 확인하기 위한 1회성 서버
	go getTokenServe()

	// Token 입력 받는 로직
	fmt.Scanln(&token)
	for !askForConfirmation("Token을 올바르게 입력 하셨나요?") {
		printGetTokenNotice()
		fmt.Scanln(&token)
	}

	// postLists 받는 부분
	postLists, blogName := getPostLists(token)

	// Create Directory
	os.MkdirAll("./result/"+blogName+"/image/", os.ModePerm)

	work := new(sync.WaitGroup)
	work.Add(len(postLists))
	for _, post := range postLists {
		go getPostRead(token, blogName, post, work)
	}
	work.Wait()
	// getPostRead(token, blogName, "106", work)

	color.Notice.Println("엔터를 입력 하시면 프로그램이 종료됩니다.")
	fmt.Scanln()
}

func getTokenServe() {
	color.Info.Prompt(redirectURL + "를 통하여 서버가 실행 되었습니다.")
	printGetTokenNotice()
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		code, ok := req.URL.Query()["code"]
		if !ok || len(code[0]) < 1 {
			log.Println("Url Param 'code' is missing")
			fmt.Fprintf(res, "error")
			return
		}
		URL := "https://www.tistory.com/oauth/access_token?" +
			"client_id=" + clientID +
			"&client_secret=" + secretKey +
			"&redirect_uri=" + redirectURL +
			"&code=" + code[0] +
			"&grant_type=" + "authorization_code"
		resGet, err := http.Get(URL)
		checkRes(resGet, err)

		bodyBytes, err := ioutil.ReadAll(resGet.Body)
		checkErr(err)

		t, err := template.ParseFiles("./http/index.html")
		checkErr(err)
		t.Execute(res, strings.Split(string(bodyBytes), "=")[1])
	})

	http.ListenAndServe(":"+port, nil)
}

func getPostLists(token string) ([]string, string) {
	var postLists []string

	// API를 호출하여 사용자의 BLOG Name, Post Count를 가져옴
	URL := "https://www.tistory.com/apis/blog/info?access_token=" + token + "&output=xml"
	res, err := undercoverGet(URL)
	checkRes(res, err)
	doc, err := goquery.NewDocumentFromReader(res.Body)
	blogName := doc.Find("tistory item blogs name").Text()
	postCount := doc.Find("tistory item blogs statistics post").Text()

	// totalPages
	postCountI, err := strconv.Atoi(postCount)
	checkErr(err)
	totalPages := postCountI / 10
	if postCountI%10 != 0 {
		totalPages++
	}

	// Go Routine를 사용하기 위한 채널
	chanPostList := make(chan []string)
	// 비동기로 PostList 가져옴
	for i := 1; i <= totalPages; i++ {
		go getPostList(token, blogName, strconv.Itoa(i), chanPostList)
	}
	// chan으로 들어온 응답값을 합치는 로직
	for i := 1; i <= totalPages; i++ {
		postLists = append(postLists, (<-chanPostList)...)
	}

	defer color.Notice.Prompt("Blog Name: " + blogName + "\nPostLists Length: " + strconv.Itoa(len(postLists)))

	return postLists, blogName
}

func getPostList(token, blogName, page string, c chan<- []string) {
	var postLists []string
	URL := "https://www.tistory.com/apis/post/list?access_token=" + token +
		"&output=" + "xml" +
		"&blogName=" + blogName +
		"&page=" + page

	res, err := undercoverGet(URL)
	checkRes(res, err)

	doc, err := goquery.NewDocumentFromReader(res.Body)

	doc.Find("tistory item posts id").Each(func(i int, s *goquery.Selection) {
		postLists = append(postLists, s.Text())
	})

	c <- postLists
}

func getPostRead(token, blogName, postID string, work *sync.WaitGroup) {
	URL := "https://www.tistory.com/apis/post/read?" +
		"access_token=" + token +
		"&blogName=" + blogName +
		"&postId=" + postID

	res, err := undercoverGet(URL)
	checkRes(res, err)

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	// Parsing title
	title := doc.Find("tistory item title").Text()
	slogan := doc.Find("tistory item slogan").Text()

	// Parsing content(html)
	content := doc.Find("tistory item content").Text()
	// Parsing date
	t, err := time.Parse("2006-01-02 15:04:05", doc.Find("tistory item date").Text())
	checkErr(err)
	date := t.Format("2006-01-02 15:04:05")
	// Parsing tags
	var tags string
	doc.Find("tistory item tags tag").Each(func(i int, s *goquery.Selection) {
		tags += `"` + s.Text() + `"`
		if doc.Find("tistory item tags tag").Length()-1 != i {
			tags += ", "
		}
	})
	frontmatter := "---\n" +
		"title: \"" + title + "\"\n" +
		"date: " + date + "\n" +
		"tags: [" + tags + "]\n" +
		"draft: false\n" +
		"---\n"

	// CreateDir
	os.MkdirAll("./result/"+blogName+"/image/"+slogan, os.ModePerm)

	markdown := convertHTMLToMd(content, blogName, slogan)

	f, err := os.Create("./result/" + blogName + "/" + slogan + ".md")
	checkErr(err)
	defer f.Close()

	w := bufio.NewWriter(f)
	_, err = w.WriteString(frontmatter + markdown)
	checkErr(err)
	w.Flush()
	defer color.Success.Prompt("Save Post Done! : " + title)
	defer work.Done()
}

func convertHTMLToMd(html, blogName, title string) string {
	converter := md.NewConverter("", true, nil)
	re := regexp.MustCompile(
		`(\[##\_Image)(\|[\w@\/.]*)(\|\w*)(\|)([\w-=" \.]*)(\|_##])`)
	replacedHTML := re.ReplaceAllStringFunc(html, func(s string) string {
		parts := strings.Split(s, "|")

		imageURL := "https://blog.kakaocdn.net/dn/" + strings.Split(parts[1], "@")[1]
		fileName := strings.Split(strings.Split(parts[1], "@")[1], "/")[2] + strings.Split(strings.Split(parts[1], "@")[1], "/")[3]
		path := "./result/" + blogName + "/image/" + title + "/" + fileName
		saveImage(imageURL, path)
		return "<img src='./image/" + title + "/" + fileName + "' />"
	})

	re = regexp.MustCompile(
		`(<div\s*class="imageblock\s*dual" ([\w\s="-:;]*>))(<table([\w\s="-:;]*)>)(<tr([\w\s="-:;]*)>)(<td>)(<img([\w\s="-:;@]*>))(<p([\w\s="-:;]*>))([\w\s="-:;]*)(<\/p>)(<\/td>)(<td>)(<a([\w\s="-:;ㄱ-ㅎ|ㅏ-ㅣ|가-힣]*>))(<img([\w\s="-:;@?]*>))(([\w\s="-:;]*)<\/a>)(<\/td>)(<\/tr>)(<\/table>(<\/div>))`)
	replacedHTML = re.ReplaceAllStringFunc(replacedHTML, func(s string) string {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
		checkErr(err)
		src, _ := doc.Find("img").First().Attr("src")
		if strings.Split(src, ".")[0] == "http://cfs" {
			imageURL := "https://blog.kakaocdn.net/dn/" + strings.Split(src, "@")[1]
			fileName := strings.Split(strings.Split(src, "@")[1], "/")[2] + strings.Split(strings.Split(src, "@")[1], "/")[3]
			path := "./result/" + blogName + "/image/" + title + "/" + fileName
			saveImage(imageURL, path)
			return "<img src='./image/" + title + "/" + fileName + "' />"
		}
		if strings.Split(src, ".")[0] == "http://kage" {
			imageURL := strings.Split(doc.Find(".cap1").Text(), `"`)[1]
			if strings.Split(imageURL, ":")[0] == "https" || strings.Split(imageURL, ":")[0] == "http" {
				fileName := strings.Split(imageURL, "/")[6] + strings.Split(imageURL, "/")[7]
				path := "./result/" + blogName + "/image/" + title + "/" + fileName
				saveImage(imageURL, path)
				return "<img src='./image/" + title + "/" + fileName + "' />"
			}
			color.Warn.Println("예외 문자(첨부파일로 예상됨..)")
			fmt.Println(s)
			return ""
		}
		imageURL := src
		if strings.Split(imageURL, ":")[0] == "https" || strings.Split(imageURL, ":")[0] == "http" {
			fmt.Println(imageURL)
			fileName := strings.Split(imageURL, "/")[6] + strings.Split(imageURL, "/")[7]
			path := "./result/" + blogName + "/image/" + title + "/" + fileName
			saveImage(imageURL, path)
			return "<img src='./image/" + title + "/" + fileName + "' />"
		}
		color.Warn.Println("??!! 처음보는 케이스")
		fmt.Println(s)
		return "<img src='" + imageURL + "' />"
	})

	markdown, err := converter.ConvertString(replacedHTML)
	checkErr(err)

	return markdown
}

func saveImage(URL, path string) {
	defer color.Info.Prompt("Save Image... : " + URL + "\n" + path)
	response, err := http.Get(URL)
	checkErr(err)
	defer response.Body.Close()

	//open a file for writing
	file, err := os.Create(path)
	checkErr(err)
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	checkErr(err)
}

func checkRes(res *http.Response, err error) {
	checkErr(err)
	checkStatus(res)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkStatus(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with Status:", res.StatusCode)
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		color.Notice.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

// Token 안내 메세지
func printGetTokenNotice() {
	color.Error.Println("*중요*")
	color.Primary.Println("https://www.tistory.com/oauth/authorize?client_id=" + clientID +
		"&redirect_uri=" + redirectURL + "&response_type=code")
	color.Notice.Println("URL로 들어가 Tistory 인증후 화면에 나오는 AccessToken를 입력해주세요!")
}

func undercoverGet(URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", URL, nil)
	checkErr(err)

	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.135 Safari/537.36")
	client := &http.Client{}
	resp, err := client.Do(req)

	return resp, err
}
