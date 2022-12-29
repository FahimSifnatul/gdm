package main

import (
	"flag"
	"gdm/app"
	"gdm/model"
	"log"
)

func main() {
	subCmd := &model.SubCmd{}
	flag.StringVar(&subCmd.FileUrl, "u", "", "file download url")
	flag.IntVar(&subCmd.ConcurrencyCount, "c", 1, "concurrent downloader count")
	flag.StringVar(&subCmd.Location, "l", "", "location to save file")
	flag.StringVar(&subCmd.FileName, "n", "", "file name to save")
	flag.Parse()

	app.DownloadManager(subCmd)
	log.Println("download is successful...")
}
