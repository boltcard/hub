package main

import (
	"card/build"
	"card/db"
	"card/util"
	"card/web"
	"database/sql"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func main() {

	var sql_db *sql.DB

	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "2006-01-02 15:04:05.999 -0700"
	Formatter.FullTimestamp = true
	Formatter.ForceColors = true
	log.SetFormatter(Formatter)

	log.Info("build version : ", build.Version)
	log.Info("build date : ", build.Date)
	log.Info("build time : ", build.Time)

	// open the database
	log.Info("init database")
	sql_db, err := sql.Open("sqlite3", "/card_data/cards.db?"+
		"_journal=WAL&"+
		"_timeout=5000&"+
		"_cache_size=10000&"+
		"_temp_store=memory&"+
		"_foreign_keys=1")
	util.CheckAndPanic(err)
	defer db.Close(sql_db)
	db.Db_init(sql_db)

	// set database connection pool parameters
	sql_db.SetMaxOpenConns(10)
	sql_db.SetMaxIdleConns(5)
	sql_db.SetConnMaxLifetime(time.Hour)
	sql_db.SetConnMaxIdleTime(15 * time.Minute)

	// check for command line arguments
	args := os.Args[1:] // without program name
	if len(args) > 0 {
		processArgs(sql_db, args)
		return
	}

	// load the web templates into memory
	log.Info("init templates")
	web.InitTemplates()

	log.Info("card service starting")

	// start the app
	app := web.NewApp(sql_db)
	router := app.SetupRoutes()

	server := &http.Server{
		Handler:      router,
		Addr:         ":8000",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
