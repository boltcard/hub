package main

import (
	"card/build"
	"card/db"
	"card/lnurlw"
	"card/pos_api"
	"card/wallet_api"
	"card/web"
	"card/web/admin"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func dumpRequest(w http.ResponseWriter, req *http.Request) {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Info(err.Error())
	} else {
		log.Info(string(requestDump))
	}
}

func main() {

	log.Info("build version : ", build.Version)
	log.Info("build date : ", build.Date)
	log.Info("build time : ", build.Time)

	if db.Db_get_setting("log_level") == "debug" {
		log.SetLevel(log.DebugLevel)
		log.Info("log level : debug")
	}

	log.Info("card service started")

	// ensure a database is available
	log.Info("init database")
	db.Db_init()

	// load the web templates into memory
	log.Info("init templates")
	web.InitTemplates()

	var router = mux.NewRouter()

	// QR code for connecting BoltCardWallet

	// web pages
	router.Path("/").Methods("GET").HandlerFunc(web.Index)
	router.Path("/favicon.ico").Methods("GET").HandlerFunc(web.Blank)

	// websocket
	router.Path("/websocket").HandlerFunc(web.WebsocketHandler)

	// admin dashboard
	router.PathPrefix("/admin/").HandlerFunc(admin.Admin)
	router.PathPrefix("/dist/").HandlerFunc(admin.Admin)

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

	// Bolt Card interface (hit from PoS when a card is tapped)
	router.Path("/ln").Methods("GET").HandlerFunc(lnurlw.LnurlwRequest)
	router.Path("/cb").Methods("GET").HandlerFunc(lnurlw.LnurlwCallback)

	// for PoS which uses part of an LndHub API
	// lndhub://a:b@https://somedomain/pos/
	router.Path("/pos/getinfo").Methods("GET").HandlerFunc(pos_api.GetInfo)
	router.Path("/pos/auth").Methods("POST").HandlerFunc(pos_api.Auth)
	router.Path("/pos/addinvoice").Methods("POST").HandlerFunc(pos_api.AddInvoice)
	router.Path("/pos/getuserinvoices").Methods("GET").HandlerFunc(pos_api.GetUserInvoices)

	router.NotFoundHandler = http.HandlerFunc(dumpRequest)

	server := &http.Server{
		Handler:      router,
		Addr:         ":8000",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
