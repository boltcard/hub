package web

import (
	"card/util"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// https://andrew-mccall.com/blog/2022/06/create-a-template-cache-for-a-go-application/

var templates = make(map[string]*template.Template)

func InitTemplates() {
	//iterate the filesystem from /web-content looking for *.html filenames

	err := filepath.WalkDir("/web-content/", visit)
	util.Check(err)
}

func visit(path string, di fs.DirEntry, err error) error {

	template_full_name := path
	template_full_name = strings.Replace(template_full_name, "/web-content/", "/", 1)

	template_names := strings.Split(template_full_name, "/")
	template_name := template_names[len(template_names)-1]

	if strings.HasSuffix(template_name, ".html") {
		// load into template cache
		ts, err := template.New(template_name).ParseFiles(path)
		util.Check(err)
		templates[template_full_name] = ts
	}

	return nil
}

func renderTemplate(w http.ResponseWriter, template_full_name string, data interface{}) {

	t, ok := templates[template_full_name]
	if !ok {
		log.Info("template not found : ", template_full_name)
		return
	}

	template_names := strings.Split(template_full_name, "/")
	template_name := template_names[len(template_names)-1]

	err := t.ExecuteTemplate(w, template_name, data)
	util.Check(err)
}
