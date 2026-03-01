package web

import (
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
	if err != nil {
		log.Error("template walk error: ", err)
	}
}

func visit(path string, di fs.DirEntry, err error) error {

	template_full_name := path
	template_full_name = strings.Replace(template_full_name, "/web-content/", "/", 1)

	template_names := strings.Split(template_full_name, "/")
	template_name := template_names[len(template_names)-1]

	// load the template cache
	if strings.HasSuffix(template_name, ".html") {
		ts, err := template.New(template_name).ParseFiles(path)
		if err != nil {
			log.Error("template parse error: ", err)
			return err
		}
		templates[template_full_name] = ts
	}

	return nil
}

func RenderHtmlFromTemplate(w http.ResponseWriter, template_full_name string, data interface{}) {

	w.Header().Add("Content-Type", "text/html")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")

	t, ok := templates[template_full_name]
	if !ok {
		log.Info("template not found : ", template_full_name)
		return
	}

	template_names := strings.Split(template_full_name, "/")
	template_name := template_names[len(template_names)-1]

	err := t.ExecuteTemplate(w, template_name, data)
	if err != nil {
		log.Error("template execution error: ", err)
	}
}

func RenderStaticContent(w http.ResponseWriter, request string) {
	cleanPath := filepath.Clean(request)
	fullPath := filepath.Join("/web-content", cleanPath)
	if !strings.HasPrefix(fullPath, "/web-content/") {
		Blank(w, nil)
		return
	}

	content, err := os.Open(fullPath)

	if err != nil {
		log.Info(err.Error())
		Blank(w, nil)
		return
	}

	defer content.Close()

	switch {
	case strings.HasSuffix(request, ".js"):
		w.Header().Add("Content-Type", "application/javascript")
	case strings.HasSuffix(request, ".css"):
		w.Header().Add("Content-Type", "text/css")
	case strings.HasSuffix(request, ".png"):
		w.Header().Add("Content-Type", "image/png")
	case strings.HasSuffix(request, ".jpg"):
		w.Header().Add("Content-Type", "image/jpeg")
	case strings.HasSuffix(request, ".map"):
		w.Header().Add("Content-Type", "application/json")
	case strings.HasSuffix(request, ".svg"):
		w.Header().Add("Content-Type", "image/svg+xml")
	case strings.HasSuffix(request, ".woff2"):
		w.Header().Add("Content-Type", "font/woff2")
	case strings.HasSuffix(request, ".ico"):
		w.Header().Add("Content-Type", "image/x-icon")
	default:
		log.Info("suffix not recognised : ", request)
		return
	}

	io.Copy(w, content)
}
