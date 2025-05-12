package main

import (
	"flag"
	"fmt"
	"os"

	"inaba.kiyuri.ca/2025/convind/data"
	"inaba.kiyuri.ca/2025/convind/wiki"
)

func main() {
	var dataStorePath string
	flag.StringVar(&dataStorePath, "data-store", "", "path to root of FS data store")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		fmt.Fprint(os.Stderr, "expected an id\n")
		os.Exit(1)
	}
	idRaw := flag.Arg(0)
	id := new(data.ID)
	err := id.UnmarshalText([]byte(idRaw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid id: %s\n", err)
		os.Exit(1)
	}

	dataStore := data.NewFSDataStoreFromSubdirectory(dataStorePath)
	data, err := dataStore.GetDataByID(*id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get data: %s\n", err)
		os.Exit(1)
	}
	page := wiki.Page{data}
	pr, err := page.LatestRevision()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get latest revision: %s\n", err)
		os.Exit(1)
	}
	rendered, err := pr.View()
	if err != nil {
		fmt.Fprintf(os.Stderr, "render latest page revision: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(rendered)
}
