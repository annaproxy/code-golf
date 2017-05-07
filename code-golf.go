package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

const base = "views/base.html"

var holes = make(map[string]*template.Template)
var index = template.Must(template.ParseFiles(base, "views/index.html"))
var minfy = minify.New()

func init() {
	minfy.AddFunc("text/html", html.Minify)

	holes["fizz-buzz"] = template.Must(
		template.ParseFiles(base, "views/fizz-buzz.html"))
}

func codeGolf(w http.ResponseWriter, r *http.Request) {
	// Skip over the initial forward slash.
	switch path := r.URL.Path[1:]; path {
	case "":
		w.Header().Set("Strict-Transport-Security", headerHSTS)
		render(w, index, map[string]interface{}{})
	case "callback":
		user := githubAuth(r.FormValue("code"))

		// TODO Set a session cookie if user.ID != 0
		fmt.Printf("%v\n", user)

		http.Redirect(w, r, "/", 302)
	case "roboto-v16":
		w.Header().Set("Cache-Control", "max-age=9999999,public")
		w.Header().Set("Content-Type", "font/woff2")
		w.Write(roboto)
	case "roboto-mono-v4":
		w.Header().Set("Cache-Control", "max-age=9999999,public")
		w.Header().Set("Content-Type", "font/woff2")
		w.Write(robotoMono)
	case cssHash:
		w.Header().Set("Cache-Control", "max-age=9999999,public")
		w.Header().Set("Content-Type", "text/css")

		if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
			w.Header().Set("Content-Encoding", "br")
			w.Write(cssBr)
		} else {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(cssGzip)
		}
	case jsHash:
		w.Header().Set("Cache-Control", "max-age=9999999,public")
		w.Header().Set("Content-Type", "application/javascript")

		if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
			w.Header().Set("Content-Encoding", "br")
			w.Write(jsBr)
		} else {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(jsGzip)
		}
	default:
		var hole, lang string

		if i := strings.IndexByte(path, '/'); i != -1 {
			hole = path[:i]
			lang = path[i+1:]
		} else {
			hole = path
		}

		if tmpl, ok := holes[hole]; ok {
			switch lang {
			case "javascript", "perl", "perl6", "php", "python", "ruby":
				vars := map[string]interface{}{"r": r}

				vars["lang"] = lang

				if r.Method == http.MethodPost {
					vars["code"] = r.FormValue("code")
					vars["pass"] = fizzBuzzAnswer == runCode(lang, r.FormValue("code"))
				} else {
					vars["code"] = examples[lang]
				}

				render(w, tmpl, vars)
			default:
				http.Redirect(w, r, "/"+hole+"/perl", 302)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func render(w http.ResponseWriter, tmpl *template.Template, vars map[string]interface{}) {
	vars["cssHash"] = cssHash
	vars["jsHash"] = jsHash

	w.Header().Set("Content-Type", "text/html;charset=utf8")

	pipeR, pipeW := io.Pipe()

	go func() {
		defer pipeW.Close()

		if err := tmpl.Execute(pipeW, vars); err != nil {
			panic(err)
		}
	}()

	if err := minfy.Minify("text/html", w, pipeR); err != nil {
		panic(err)
	}
}

func runCode(lang, code string) string {
	var out bytes.Buffer

	// TODO Need better IDs
	// TODO Use github.com/opencontainers/runc/libcontainer
	cmd := exec.Cmd{
		Args:   []string{"runc", "start", "id"},
		Dir:    "containers/" + lang,
		Path:   "/usr/bin/runc",
		Stdin:  strings.NewReader(code),
		Stdout: &out,
	}

	if err := cmd.Run(); err != nil {
		println(err.Error())
	}

	return string(bytes.TrimSpace(out.Bytes()))
}
