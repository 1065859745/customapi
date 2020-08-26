package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

var port = flag.String("p", "8018", "Port of serive")
var passwd = flag.String("P", "", "The password of request authorization in this API")

func sendMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authParam := strings.Trim(fmt.Sprint(r.Header["Authorization"]), "[]")
	matched, _ := regexp.MatchString(`key=\w+`, authParam)
	if !matched {
		log.Println(r.Host + "-Authorized faild")
		http.Error(w, "Authorized faild", http.StatusUnauthorized)
		return
	}
	// existed key
	// authorization key
	if strings.TrimLeft(authParam, "key=") != *passwd {
		log.Println(r.Host + "-Authorized faild")
		http.Error(w, "Authorized faild", http.StatusUnauthorized)
		return
	}
	phones := r.URL.Query().Get("phones")
	messages := r.URL.Query().Get("messages")
	if messages == "" || len(messages) >= 64 {
		log.Println(r.Host + "-Messages is null or more than 65bytes")
		http.Error(w, "Messages is null or more than 65bytes", http.StatusBadRequest)
		return
	}
	if phones == "" {
		log.Println(r.Host + "-No phones parameters")
		http.Error(w, "No phones parameters", http.StatusBadRequest)
		return
	}
	phonesArr := strings.Fields(phones)
	for i, v := range phonesArr {
		if matched, _ = regexp.MatchString(`\d{11}`, v); !matched {
			log.Printf(r.Host+"-Phone[%d] is not a 11 digit phone number\n", i)
			http.Error(w, "Phone number not a 11 digit", http.StatusBadRequest)
			return
		}
	}

	cmd := exec.Command("java", "Send", phones)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Println(r.Host + "-Pipe write error")
		http.Error(w, "1", http.StatusInternalServerError)
		return
	}
	go (func() {
		defer stdin.Close()
		io.WriteString(stdin, messages)
	})()
	err = cmd.Run()
	if err != nil {
		log.Println(r.Host + "-Excured error")
		http.Error(w, "1", http.StatusInternalServerError)
		return
	}
	log.Printf("Send messages: %s \nTo: %s", messages, fmt.Sprint(phonesArr))
	fmt.Fprint(w, "0")
}

func homeTip(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	const tip = `请求类型: GET
请求参数: {phones: {类型: string, 是否必须: 是, 备注: 11位手机号}, msg: {类型: string, 是否必须: 是, 备注: 字数小于64}}
curl请求示例: curl --header "Authorization: key=aaaaa" http://{{.Host}}/sendMsg?phones="1312xxxxxxx 15600xxxxxx 147939xxxxx"&messages="Hello"
响应示例: {0: 发送成功, 1: 发送失败}`
	homeTemp := template.Must(template.New("").Parse(tip))
	var v = struct {
		Host string
	}{
		r.Host,
	}
	homeTemp.Execute(w, &v)
}

func main() {
	flag.Parse()
	http.HandleFunc("/sendMsg", sendMsg)
	http.HandleFunc("/", homeTip)
	log.Println("API service will start at localhost:" + *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
