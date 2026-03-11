package main

import (
	"card/build"
	"card/db"
	"card/phoenix"
	"card/web"
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// open the database — two connections for read/write separation
	log.Info("init database")
	dsn := "/card_data/cards.db?" +
		"_journal=WAL&" +
		"_synchronous=FULL&" + // ensure commits survive power loss
		"_timeout=5000&" + // 5 second timeout for busy
		"_cache_size=10000&" + // 5x more memory for caching pages
		"_temp_store=memory&" +
		"_foreign_keys=1&" +
		"_secure_delete=1&" + // overwrite deleted data
		"_auto_vacuum=INCREMENTAL" // prevent file bloat

	// write connection: single conn serialises all writes via Go's pool
	writeDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal("failed to open write database: ", err)
	}
	writeDB.SetMaxOpenConns(1)
	defer db.Close(writeDB)

	// read connection: many concurrent readers (WAL allows this)
	readDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal("failed to open read database: ", err)
	}
	defer readDB.Close()

	db.Db_init(writeDB)

	// set log level from database setting
	logLevel := db.Db_get_setting(readDB, "log_level")
	if level, err := log.ParseLevel(logLevel); err == nil {
		log.SetLevel(level)
		log.Info("log level set to ", logLevel)
	}

	// pre-load phoenix credentials
	if err := phoenix.InitPassword(); err != nil {
		log.Warn("phoenix config not available at startup: ", err)
	}

	// check for command line arguments
	args := os.Args[1:] // without program name
	if len(args) > 0 {
		processArgs(writeDB, args)
		return
	}

	// load the web templates into memory
	log.Info("init templates")
	web.InitTemplates()

	log.Info("card service starting")

	// start the app
	app := web.NewApp(readDB, writeDB)
	router := app.SetupRoutes()

	server := &http.Server{
		Handler:      router,
		Addr:         ":8000",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	// graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	log.Info("card service started")

	<-stop
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("shutdown error: ", err)
	}

	log.Info("shutdown complete")
}
