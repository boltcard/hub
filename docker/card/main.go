package main

import (
	"card/bcp"
	"card/build"
	"card/db"
	"card/lnurlw"
	"card/pos_api"
	"card/util"
	"card/wallet_api"
	"card/web"
	"card/web/admin"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func main() {

	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "2006-01-02 15:04:05.999 -0700"
	Formatter.FullTimestamp = true
	Formatter.ForceColors = true
	log.SetFormatter(Formatter)

	log.Info("build version : ", build.Version)
	log.Info("build date : ", build.Date)
	log.Info("build time : ", build.Time)

	// https://goperf.dev/01-common-patterns/gc/#memory-limiting-with-gomemlimit
	// to avoid occasional container termination by docker OOM killer
	// docker-compose is set up to restart but this could still cause some downtime
	// also ensure memory is regularly freed to the OS
	debug.SetMemoryLimit(2 << 27) // 256 Mb
	go util.FreeMemory()

	if db.Db_get_setting("log_level") == "debug" {
		log.SetLevel(log.DebugLevel)
		log.Info("log level : debug")
	}

	// check for command line arguments
	args := os.Args[1:] // without program name
	if len(args) > 0 {
		processArgs(args)
		return
	}

	log.Info("card service started")

	// ensure a database is available
	log.Info("init database")
	db.Db_init()

	// load the web templates into memory
	log.Info("init templates")
	web.InitTemplates()

	var router = mux.NewRouter()

	// status monitoring
	router.Path("/").Methods("HEAD").HandlerFunc(web.StatusResponse)

	// web pages
	router.Path("/").Methods("GET").HandlerFunc(web.HomePage)
	router.Path("/favicon.ico").Methods("GET").HandlerFunc(web.Blank)
	router.Path("/balance/").Methods("GET").HandlerFunc(web.BalancePage)

	// AJAX
	router.Path("/balance-ajax").Methods("GET").HandlerFunc(web.BalanceAjaxPage)

	// websocket
	router.Path("/websocket").HandlerFunc(web.WebsocketHandler)

	// admin dashboard
	router.PathPrefix("/admin/").HandlerFunc(admin.Admin)
	router.PathPrefix("/dist/").HandlerFunc(admin.Admin)

	// for Bolt Card Programmer app
	router.Path("/new").Methods("GET").HandlerFunc(bcp.CreateCard)
	router.Path("/batch").Methods("POST").HandlerFunc(bcp.BatchCreateCard)

	// Bolt Card interface (hit from PoS when a card is tapped)
	router.Path("/ln").Methods("GET").HandlerFunc(lnurlw.LnurlwRequest)
	router.Path("/cb").Methods("GET").HandlerFunc(lnurlw.LnurlwCallback)

	if db.Db_get_setting("bolt_card_hub_api") == "enabled" {
		// BoltCardHub API
		// LNDHUB API reference https://github.com/BlueWallet/LndHub/blob/master/doc/Send-requirements.md
		router.Path("/getinfobolt").Methods("GET").HandlerFunc(wallet_api.GetInfoBolt)
		router.Path("/create").Methods("POST").HandlerFunc(wallet_api.Create)
		router.Path("/auth").Methods("POST").HandlerFunc(wallet_api.Auth)
		router.Path("/getbtc").Methods("GET").HandlerFunc(wallet_api.GetBtc) // Get user's BTC address to top-up his account
		router.Path("/balance").Methods("GET").HandlerFunc(wallet_api.Balance)
		router.Path("/gettxs").Methods("GET").HandlerFunc(wallet_api.GetTxs)         // /gettxs?limit=10&offset=0 (onchain & lightning)
		router.Path("/getpending").Methods("GET").HandlerFunc(wallet_api.GetPending) // for onchain txs only
		router.Path("/getuserinvoices").Methods("GET").HandlerFunc(wallet_api.GetUserInvoices)
		router.Path("/getcardkeys").Methods("POST").HandlerFunc(wallet_api.GetCardKeys) // creating a new card
		router.Path("/addinvoice").Methods("POST").HandlerFunc(wallet_api.AddInvoice)
		router.Path("/payinvoice").Methods("POST").HandlerFunc(wallet_api.PayInvoice)
		router.Path("/getcard").Methods("POST").HandlerFunc(wallet_api.GetCard)   // get card details
		router.Path("/wipecard").Methods("POST").HandlerFunc(wallet_api.WipeCard) // return keys and deactivate card
		router.Path("/updatecardwithpin").Methods("POST").HandlerFunc((wallet_api.UpdateCardWithPin))
	}

	if db.Db_get_setting("bolt_card_pos_api") == "enabled" {
		// for PoS which uses part of an LndHub API
		// lndhub://a:b@https://somedomain/pos/
		router.Path("/pos/getinfo").Methods("GET").HandlerFunc(pos_api.GetInfo)
		router.Path("/pos/auth").Methods("POST").HandlerFunc(pos_api.Auth)
		router.Path("/pos/addinvoice").Methods("POST").HandlerFunc(pos_api.AddInvoice)
		router.Path("/pos/getuserinvoices").Methods("GET").HandlerFunc(pos_api.GetUserInvoices)
	}

	// router.NotFoundHandler = http.HandlerFunc(web.DumpRequest)

	server := &http.Server{
		Handler:      router,
		Addr:         ":8000",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
