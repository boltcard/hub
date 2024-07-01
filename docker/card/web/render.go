package web

import (
	"card/phoenix"
	"card/util"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

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

func renderContent(w http.ResponseWriter, request string) {

	// default to index.html
	if strings.HasSuffix(request, "/") {
		log.Info("page : ", request)
		request = request + "index.html"
	}

	// only log page requests
	if strings.HasSuffix(request, ".html") {
		template_path := strings.Replace(request, "/admin/", "/dist/pages/admin/", 1)
		w.Header().Add("Content-Type", "text/html")

		// HACK: to test template data injection
		if request == "/admin/index.html" {

			balance, err := phoenix.GetBalance()
			if err != nil {
				log.Warn("phoenix error: ", err.Error())
			}

			info, err := phoenix.GetInfo()
			if err != nil {
				log.Warn("phoenix error: ", err.Error())
			}

			totalInboundSats := 0
			for _, channel := range info.Channels {
				totalInboundSats += channel.InboundLiquiditySat
			}

			// https://gosamples.dev/print-number-thousands-separator/
			// https://stackoverflow.com/questions/11123865/format-a-go-string-without-printing
			p := message.NewPrinter(language.English)
			FeeCreditSatStr := p.Sprintf("%d sats", balance.FeeCreditSat)
			BalanceSatStr := p.Sprintf("%d sats", balance.BalanceSat)
			ChannelsStr := p.Sprintf("%d", len(info.Channels))
			TotalInboundSatsStr := p.Sprintf("%d sats", totalInboundSats)

			data := struct {
				FeeCredit string
				Balance   string
				Channels  string
				Inbound   string
			}{
				FeeCredit: FeeCreditSatStr,
				Balance:   BalanceSatStr,
				Channels:  ChannelsStr,
				Inbound:   TotalInboundSatsStr,
			}

			renderTemplate(w, template_path, data)
			return
		}

		renderTemplate(w, template_path, nil)
		return
	}

	// everything except .html
	content, err := os.Open("/web-content" + request)

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
