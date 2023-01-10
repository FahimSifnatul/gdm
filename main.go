package main

import (
	"flag"
	"fmt"

	"gdm/app"
	"gdm/model"
)

const (
	bold  string = "\033[1m"
	green string = "\033[92m"
)

func main() {
	subCmd := &model.SubCmd{}
	flag.StringVar(&subCmd.FileUrl, "u", "", "file download url")
	flag.IntVar(&subCmd.ConcurrencyCount, "c", 1, "concurrent downloader count")
	flag.StringVar(&subCmd.Location, "l", "", "location to save file")
	flag.StringVar(&subCmd.FileName, "n", "", "file name to save")
	flag.Parse()

	app.DownloadManager(subCmd)
	fmt.Printf("%s%sDownload Completed!!!\n", bold, green)
}
