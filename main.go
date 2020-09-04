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
	Name    string
	Require bool
	Pattern string
	Tip     string
}

var port = flag.String("p", "8018", "Port of serive")
var configFile = flag.String("-config.file", "main.json", "Path of configure file")

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

	tip, _ := ioutil.ReadFile("homeTip")
	homeTemp := template.Must(template.New("").Parse(string(tip)))
	var v = struct {
		Host string
	}{
		r.Host,
	}
	homeTemp.Execute(w, &v)
}

func middleWare(configs *config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check method.
		if r.Method != configs.Method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Check password.
		if configs.Pwd != "" {
			authParam := strings.Trim(fmt.Sprint(r.Header["Authorization"]), "[]")
			matched, _ := regexp.MatchString(`key=\w+`, authParam)
			if !matched {
				log.Println(r.Host + " - Authorized faild")
				http.Error(w, "Authorized faild", http.StatusUnauthorized)
				return
			}
			// existed key.
			// authorization key.
			if strings.TrimLeft(authParam, "key=") != configs.Pwd {
				log.Println(r.Host + " - Authorized faild")
				http.Error(w, "Authorized faild", http.StatusUnauthorized)
				return
			}
		}

		// Check parameters.
		query := r.URL.Query()
		for _, v := range configs.Parameters {
			param := query.Get(v.Name)
			if param == "" {
				if v.Require {
					http.Error(w, v.Name+" was required", http.StatusBadRequest)
					return
				}
				break
			}
			if matched, _ := regexp.MatchString(v.Pattern, param); !matched {
				log.Printf("%s %s - Parameter %s error.\n", r.Host, r.URL.Path, param)
				http.Error(w, fmt.Sprintf("Tips of parameter %s: %s", v.Name, v.Tip), http.StatusBadRequest)
				return
			}
		}

		cmd := exec.Command(configs.Commands[0], configs.Commands[1:]...)
		for i, v := range configs.Commands {
			configs.Commands[i] = achieve(v, query)
		}

		// Check pipeline.
		if configs.StdinPipe != "" {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Printf("%s %s - Pipe write error", r.Host, r.URL.Path)
				http.Error(w, "1", http.StatusInternalServerError)
				return
			}
			configs.StdinPipe = achieve(configs.StdinPipe, query)
			go (func() {
				defer stdin.Close()
				io.WriteString(stdin, configs.StdinPipe)
			})()
		}

		// Excure Command.
		if configs.Output {
			out, err := cmd.Output()
			if err != nil {
				log.Printf("%s %s - Cmd excure error", r.Host, r.URL.Path)
				http.Error(w, "1", http.StatusInternalServerError)
				return
			}
			if out == nil {
				io.WriteString(w, "0")
				return
			}
			io.WriteString(w, string(out))
		} else {
			err := cmd.Run()
			if err != nil {
				log.Printf("%s %s - Cmd excure error", r.Host, r.URL.Path)
				http.Error(w, "1", http.StatusInternalServerError)
				return
			}
			io.WriteString(w, "0")
		}
	}
}

func achieve(s string, m map[string][]string) string {
	if vArr := regexp.MustCompile(`\$\w+|\$\{\w+\}`).FindAllString(s, -1); vArr != nil {
		vArr = slice.DelStrSame(vArr)
		for _, v := range vArr {
			vTrim := strings.Trim(v, `${}`)
			for key, value := range m {
				fmt.Println(value, value[0])
				if key == vTrim {
					s = regexp.MustCompile(regexp.QuoteMeta(v)).ReplaceAllString(s, value[0])
				}
			}
		}
	}
	return s
}

func main() {
	var configs []config
	configContent, err := ioutil.ReadFile("main.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(configContent, &configs)
	if err != nil {
		log.Fatalln(err)
	}

	flag.Parse()
	for _, v := range configs {
		// Check commands.
		if len(v.Commands) == 0 {
			log.Fatalln("Args of commands was required.")
			return
		}
		http.HandleFunc(v.Path, middleWare(&v))
	}
	http.HandleFunc("/", homeTip)
	log.Printf("API service will start at localhost:%s.\n", *port)
	log.Fatalln(http.ListenAndServe(":"+*port, nil))
}
