package main

import (
	"card/build"
	"card/db"
	"card/util"
	"card/web"
	"database/sql"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func main() {

	var db_conn *sql.DB

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

	// ensure a database is available
	log.Info("init database")
	// open a database connection
	db_conn, err := sql.Open("sqlite3", "/card_data/cards.db?_journal=WAL&_timeout=5000")
	util.CheckAndPanic(err)
	defer db.Close(db_conn)
	db.Db_init(db_conn)

	// check for command line arguments
	args := os.Args[1:] // without program name
	if len(args) > 0 {
		processArgs(db_conn, args)
		return
	}

	// load the web templates into memory
	log.Info("init templates")
	web.InitTemplates()

	log.Info("card service starting")

	// start the app
	app := web.NewApp(db_conn)
	router := app.SetupRoutes()

	server := &http.Server{
		Handler:      router,
		Addr:         ":8000",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
