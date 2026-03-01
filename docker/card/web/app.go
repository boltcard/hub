package web

import (
	"card/db"
	"database/sql"

	"github.com/gorilla/mux"
)

type App struct {
	db_conn *sql.DB
}

func NewApp(db_conn *sql.DB) *App {
	return &App{db_conn: db_conn}
}

func (app *App) SetupRoutes() *mux.Router {

	var router = mux.NewRouter()

	// status monitoringStatusResponse
	router.Path("/").Methods("HEAD").HandlerFunc(app.CreateHandler_Status())

	// web pages
	router.Path("/").Methods("GET").HandlerFunc(HomePage)
	router.Path("/favicon.ico").Methods("GET").HandlerFunc(Blank)
	router.Path("/balance/").Methods("GET").HandlerFunc(BalancePage)

	// AJAX
	router.Path("/balance-ajax").Methods("GET").HandlerFunc(app.CreateHandler_BalanceAjaxPage())

	// websocket
	router.Path("/websocket").HandlerFunc(app.CreateHandler_Websocket())

	// admin API (JSON endpoints) — must be before /admin/ catch-all
	router.PathPrefix("/admin/api/").HandlerFunc(app.CreateHandler_AdminApi())

	// admin dashboard
	router.PathPrefix("/admin/").HandlerFunc(app.CreateHandler_Admin())

	// public assets that do not need authentication
	router.PathPrefix("/public/").HandlerFunc(app.CreateHandler_Public())

	// for Bolt Card Programmer app
	router.Path("/new").Methods("GET", "POST").HandlerFunc(app.CreateHandler_CreateCard())

	// Bolt Card interface (hit from PoS when a card is tapped)
	router.Path("/ln").Methods("GET").HandlerFunc(app.CreateHandler_LnurlwRequest())
	router.Path("/cb").Methods("GET").HandlerFunc(app.CreateHandler_LnurlwCallback())

	if db.Db_get_setting(app.db_conn, "bolt_card_hub_api") == "enabled" {
		// BoltCardHub API
		// LNDHUB API reference https://github.com/BlueWallet/LndHub/blob/master/doc/Send-requirements.md
		router.Path("/create").Methods("POST").HandlerFunc(app.CreateHandler_Create())
		router.Path("/auth").Methods("POST").HandlerFunc(app.CreateHandler_Auth())
		router.Path("/balance").Methods("GET").HandlerFunc(app.CreateHandler_Balance())
		router.Path("/gettxs").Methods("GET").HandlerFunc(app.CreateHandler_GetTxs())         // /gettxs?limit=10&offset=0 (onchain & lightning)
		router.Path("/getpending").Methods("GET").HandlerFunc(app.CreateHandler_GetPending()) // for onchain txs only
		router.Path("/getuserinvoices").Methods("GET").HandlerFunc(app.CreateHandler_WalletApi_GetUserInvoices())
		router.Path("/getcardkeys").Methods("POST").HandlerFunc(app.CreateHandler_WalletApi_GetCardKeys()) // creating a new card
		router.Path("/addinvoice").Methods("POST").HandlerFunc(app.CreateHandler_AddInvoice())
		router.Path("/payinvoice").Methods("POST").HandlerFunc(app.CreateHandler_WalletApi_PayInvoice())
		router.Path("/getcard").Methods("POST").HandlerFunc(app.CreateHandler_WalletApi_GetCard())   // get card details
		router.Path("/wipecard").Methods("POST").HandlerFunc(app.CreateHandler_WalletApi_WipeCard()) // return keys and deactivate card
		router.Path("/updatecardwithpin").Methods("POST").HandlerFunc(app.CreateHandler_WalletApi_UpdateCardWithPin())
	}

	if db.Db_get_setting(app.db_conn, "bolt_card_pos_api") == "enabled" {
		// for PoS which uses part of an LndHub API
		// lndhub://a:b@https://somedomain/pos/
		router.Path("/pos/getinfo").Methods("GET").HandlerFunc(app.CreateHandler_PosApi_GetInfo())
		router.Path("/pos/addinvoice").Methods("POST").HandlerFunc(app.CreateHandler_PosApi_AddInvoice())
		router.Path("/pos/getuserinvoices").Methods("GET").HandlerFunc(app.CreateHandler_PosApi_GetUserInvoices())
	}

	return router
}
