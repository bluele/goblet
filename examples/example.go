package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bluele/goblet"
	"github.com/codegangsta/negroni"
)

type Config struct {
	DBAddr  string
	LogPath string
}

type DB interface {
	Query(string) (string, error)
}

type DBImpl struct {
	Addr string
}

func (db *DBImpl) Query(q string) (string, error) {
	return q, nil
}

type Logger interface {
	Print(...interface{})
}

func main() {
	gb := goblet.New()
	gb.MustSetALL([]goblet.Def{
		{
			Name: "config",
			Constructor: func() (*Config, error) {
				return &Config{DBAddr: "127.0.0.1:6347"}, nil
			},
			Singleton: true,
		},
		{
			Name: "db",
			Constructor: func(config *Config) (DB, error) {
				return &DBImpl{config.DBAddr}, nil
			},
			Refs:      goblet.Refs{"config"},
			Singleton: true,
		},
		{
			Name: "logger",
			Constructor: func(config *Config) (Logger, error) {
				return log.New(os.Stderr, "", log.LstdFlags), nil
			},
			Refs:      goblet.Refs{"config"},
			Singleton: true,
		},
	})

	mw := gb.MustCall(func(db DB, logger Logger) (negroni.HandlerFunc, error) {
		return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			logger.Print("before:")
			next(w, r)
			logger.Print(":after")
		}, nil
	}, goblet.Refs{
		goblet.ParallelRefs("db", "logger"),
	}).(negroni.HandlerFunc)

	mux := http.NewServeMux()
	mux.HandleFunc("/",
		gb.MustCall(
			func(db DB) (http.HandlerFunc, error) {
				return func(w http.ResponseWriter, r *http.Request) {
					q, _ := db.Query(r.URL.Query().Get("q"))
					fmt.Fprint(w, q)
				}, nil
			},
			goblet.Refs{"db"},
		).(http.HandlerFunc),
	)

	n := negroni.Classic()
	n.Use(mw)
	n.UseHandler(mux)
	n.Run(":3030")
}
