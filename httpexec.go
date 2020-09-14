package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/1065859745/slice"
)

type config struct {
	Path       string
	Method     string
	Parameters []parameter
	Commands   []string
	StdinPipe  string
	Output     bool
	Pwd        string
}

type parameter struct {
	Name    string `json:"name"`
	Require bool   `json:"require"`
	Pattern string `json:"pattern"`
	Tip     string `json:"tip"`
}

type homeTip struct {
	Path       string      `json:"path"`
	Method     string      `json:"method"`
	Params     []parameter `json:"params"`
	ResExample string      `json:"resExample"`
	ReqExample string      `json:"reqExample"`
}

var port = flag.String("p", "8018", "Port of serive")
var configFile = flag.String("-config.file", "httpexec.json", "Path of configuration")
var tip []byte

func (c config) createHomeTip() homeTip {
	h := homeTip{Path: c.Path, Method: c.Method, Params: c.Parameters, ResExample: "0:success 1:failed"}
	var arr []string
	for _, v := range h.Params {
		arr = append(arr, fmt.Sprintf("%s=%s", v.Name, v.Tip))
	}
	switch c.Pwd {
	case "":
		h.ReqExample = fmt.Sprintf("curl -X %s http://{{.Host}}%s", h.Method, h.Path)
	default:
		h.ReqExample = fmt.Sprintf("curl -X %s --header \"Authorization: key=xxxxx\" \"http://{{.Host}}%s", h.Method, h.Path)
	}
	if h.Params != nil {
		h.ReqExample += fmt.Sprintf("?%s", strings.Join(arr, "&"))
	}
	h.ReqExample += `"`
	return h
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	homeTemp := template.Must(template.New("").Parse(string(tip)))
	var v = struct {
		Host string
	}{
		r.Host,
	}
	homeTemp.Execute(w, &v)
}

func middleWare(conf *config) http.HandlerFunc {
	switch conf.Method {
	case "":
		conf.Method = "GET"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		commands := conf.Commands
		pipe := conf.StdinPipe
		// Check method.
		if r.Method != conf.Method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Check password.
		if conf.Pwd != "" {
			authParam := strings.Trim(fmt.Sprint(r.Header["Authorization"]), "[]")
			matched, _ := regexp.MatchString(`key=\w+`, authParam)
			if !matched {
				log.Printf("%s [ERROR] Authorized faild", r.RemoteAddr)
				http.Error(w, "Authorized faild", http.StatusUnauthorized)
				return
			}
			// existed key.
			// authorization key.
			if strings.TrimLeft(authParam, "key=") != conf.Pwd {
				log.Printf("%s [ERROR] Authorized faild", r.RemoteAddr)
				http.Error(w, "Authorized faild", http.StatusUnauthorized)
				return
			}
		}

		// Check request parameters.
		query := r.URL.Query()
		for _, v := range conf.Parameters {
			param := query.Get(v.Name)
			if param == "" {
				if v.Require {
					http.Error(w, v.Name+" was required", http.StatusBadRequest)
					return
				}
				break
			}
			if matched, _ := regexp.MatchString(v.Pattern, param); !matched {
				log.Printf("%s %s [ERROR] Parameter %s error", r.RemoteAddr, r.URL.Path, param)
				http.Error(w, fmt.Sprintf("Tips of parameter %s: %s", v.Name, v.Tip), http.StatusBadRequest)
				return
			}
		}
		// Achieve commands.
		for i, v := range commands {
			commands[i] = achieve(v, query)
		}
		cmd := exec.Command(commands[0], commands[1:]...)
		// Check pipeline.
		if pipe != "" {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Printf("%s %s [ERROR] Pipe write error", r.RemoteAddr, r.URL.Path)
				http.Error(w, "1", http.StatusInternalServerError)
				return
			}
			pipe = achieve(pipe, query)
			go (func() {
				defer stdin.Close()
				io.WriteString(stdin, pipe)
			})()
		}
		// Excure Command.
		if conf.Output {
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("%s %s [ERROR] %s", r.RemoteAddr, r.URL.Path, err.Error())
				http.Error(w, "1", http.StatusInternalServerError)
				return
			}
			log.Printf("%s %s %s", r.RemoteAddr, r.URL.Path, query)
			fmt.Fprintf(w, "%s", out)
			return
		}
		err := cmd.Start()
		if err != nil {
			log.Printf("%s %s Cmd excure error", r.RemoteAddr, r.URL.Path)
			http.Error(w, "1", http.StatusInternalServerError)
		}
		io.WriteString(w, "0")
		log.Printf("%s %s %s", r.RemoteAddr, r.URL.Path, query)
	}
}

func achieve(s string, m map[string][]string) string {
	if vArr := regexp.MustCompile(`\$\w+|\$\{\w+\}`).FindAllString(s, -1); vArr != nil {
		vArr = slice.DelSameStr(vArr)
		for _, v := range vArr {
			vTrim := strings.Trim(v, `${}`)
			for key, value := range m {
				if key == vTrim {
					s = regexp.MustCompile(regexp.QuoteMeta(v)).ReplaceAllString(s, value[0])
				}
			}
		}
	}
	return s
}

func main() {
	flag.Parse()
	var configs []config
	content, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(content, &configs)
	if err != nil {
		log.Fatalln(err)
	}
	var array []homeTip
	for i, c := range configs {
		// Check commands.
		if c.Commands == nil {
			log.Fatalln("Commands args was required.")
		}
		// Check config parameter.
		var arr []string
		for _, p := range c.Parameters {
			if p.Name == "" {
				log.Fatalln("The name of parameter cannot be empty.")
			}
			arr = append(arr, p.Name)
		}
		if slice.IncludeSameStr(arr) {
			log.Fatalln("Parameters name cannot be the same.")
		}
		http.HandleFunc(configs[i].Path, middleWare(&configs[i]))
		array = append(array, configs[i].createHomeTip())
	}
	if tip, err = json.Marshal(array); err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/info", home)
	log.Printf("API service will start at localhost:%s.", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
