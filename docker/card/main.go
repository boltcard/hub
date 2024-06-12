package main

import (
	"card/db"
	"card/lnurlw"
	"card/pos_api"
	"card/wallet_api"
	"card/web"
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

func getInfoBolt(w http.ResponseWriter, r *http.Request) {
	log.Info("getInfoBolt request received")
	w.Write([]byte(""))
}

func getBtc(w http.ResponseWriter, r *http.Request) {
	log.Info("getBtc request received")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonData := []byte(`[{ ""}]`)
	w.Write(jsonData)
}

func getPending(w http.ResponseWriter, r *http.Request) {
	log.Info("getPending request received")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonData := []byte(`[]`) // array
	w.Write(jsonData)
}

func favIcon(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(""))
}

func main() {

	// ensure a database is available
	db.Db_init()

	// load the web templates into memory
	web.InitTemplates()

	log.Info("card service started")

	if db.Db_get_setting("log_level") == "debug" {
		log.SetLevel(log.DebugLevel)
		log.Info("log level is set to debug")
	}

	var router = mux.NewRouter()

	// QR code for connecting BoltCardWallet

	// web pages
	router.Path("/").Methods("GET").HandlerFunc(web.HomePage)
	router.Path("/favicon.ico").Methods("GET").HandlerFunc(favIcon)
	router.Path("/admin/").HandlerFunc(web.DashboardPage)

	// BoltCardHub API
	// LNDHUB API reference https://github.com/BlueWallet/LndHub/blob/master/doc/Send-requirements.md
	router.Path("/getinfobolt").Methods("GET").HandlerFunc(getInfoBolt)
	router.Path("/create").Methods("POST").HandlerFunc(wallet_api.Create)
	router.Path("/auth").Methods("POST").HandlerFunc(wallet_api.Auth)
	router.Path("/getbtc").Methods("GET").HandlerFunc(getBtc) // Get user's BTC address to top-up his account
	router.Path("/balance").Methods("GET").HandlerFunc(wallet_api.Balance)
	router.Path("/gettxs").Methods("GET").HandlerFunc(wallet_api.GetTxs) // /gettxs?limit=10&offset=0 (onchain & lightning)
	router.Path("/getpending").Methods("GET").HandlerFunc(getPending)    // for onchain txs only
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
