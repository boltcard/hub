package web

import (
	"card/util"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// https://andrew-mccall.com/blog/2022/06/create-a-template-cache-for-a-go-application/

var templates = make(map[string]*template.Template)

func InitTemplates() {
	//iterate the filesystem from /web-content looking for *.html filenames

	err := filepath.WalkDir("/web-content/", visit)
	util.CheckAndPanic(err)
}

func visit(path string, di fs.DirEntry, err error) error {

	template_full_name := path
	template_full_name = strings.Replace(template_full_name, "/web-content/", "/", 1)

	template_names := strings.Split(template_full_name, "/")
	template_name := template_names[len(template_names)-1]

	if strings.HasSuffix(template_name, ".html") {
		log.Info("loading template cache: " + template_full_name + " : from : " + path)
		// load into template cache
		ts, err := template.New(template_name).ParseFiles(path)
		util.CheckAndPanic(err)
		templates[template_full_name] = ts
	}

	return nil
}

func RenderHtmlFromTemplate(w http.ResponseWriter, template_full_name string, data interface{}) {

	w.Header().Add("Content-Type", "text/html")

	t, ok := templates[template_full_name]
	if !ok {
		log.Info("template not found : ", template_full_name)
		return
	}

	template_names := strings.Split(template_full_name, "/")
	template_name := template_names[len(template_names)-1]

	err := t.ExecuteTemplate(w, template_name, data)
	util.CheckAndPanic(err)
}

// TODO: cache these in memory
func RenderStaticContent(w http.ResponseWriter, request string) {

	content, err := os.Open("/web-content/" + request)

	if err != nil {
		log.Info(err.Error())
		Blank(w, nil)
		return
	}

	defer content.Close()

	// https://stackoverflow.com/questions/19911929/what-mime-type-should-i-use-for-javascript-source-map-files
	switch {
	case strings.HasSuffix(request, ".js"):
		w.Header().Add("Content-Type", "application/json")
	case strings.HasSuffix(request, ".css"):
		w.Header().Add("Content-Type", "text/css")
	case strings.HasSuffix(request, ".png"):
		w.Header().Add("Content-Type", "image/png")
	case strings.HasSuffix(request, ".jpg"):
		w.Header().Add("Content-Type", "image/jpeg")
	case strings.HasSuffix(request, ".map"):
		w.Header().Add("Content-Type", "application/json")
	default:
		log.Info("suffix not recognised : ", request)
		return
	}

	io.Copy(w, content)
}
