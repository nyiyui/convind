package main

import (
	"flag"
	"log"
	"net/http"

	"inaba.kiyuri.ca/2025/convind/data"
	"inaba.kiyuri.ca/2025/convind/wiki/server"
)

func main() {
	var bind string
	var dataStorePath string
	flag.StringVar(&bind, "bind", "127.0.0.1:8080", "server binds to this address")
	flag.StringVar(&dataStorePath, "data-store", "", "path to root of FS data store")
	flag.Parse()

	dataStore := data.NewFSDataStoreFromSubdirectory(dataStorePath)
	s, err := server.New(dataStore)
	if err != nil {
		panic(err)
	}
	log.Printf("listening on %s…", bind)
	log.Fatal(http.ListenAndServe(bind, s))
}
