package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gdm/model"
)

type Downloader interface {
	NewDownloader()

	// private method
	getRange(presChunk int) string

	setFileNameFromRawUrl()
	setHeadInfo()
	setFileLocation()

	createFile()
	createPartFile(partNo int) *os.File
	combinePartFiles()

	validate()
}

type downloader struct {
	FileUrl          string `flg:"u"`
	ConcurrencyCount int    `flg:"c"`
	Location         string `flg:"l"`
	FileName         string `flg:"n"`

	AcceptRanges  string
	ContentLength int
	ChunkSize     int
	File          *os.File
	WG            sync.WaitGroup
}

func DownloadManager(subCmd *model.SubCmd) {
	d := &downloader{
		FileUrl:          subCmd.FileUrl,
		ConcurrencyCount: subCmd.ConcurrencyCount,
		Location:         subCmd.Location,
		FileName:         subCmd.FileName,
	}

	// Validate downloader configuration
	d.validate()

	// Create new file
	d.createFile()
	defer d.File.Close()

	d.NewDownloader()
}

func (d *downloader) NewDownloader() {
	// Define goroutines numbers to wait in WaitGroup
	d.WG.Add(d.ConcurrencyCount)

	// concurrently download the file
	for i := 0; i < d.ConcurrencyCount; i++ {
		go func(i int) {
			defer d.WG.Done()
			client := http.Client{}

			req, err := http.NewRequest(http.MethodGet, d.FileUrl, nil)
			if err != nil {
				log.Fatal(err)
			}

			if d.ChunkSize != -1 {
				req.Header.Add("Range", d.getRange(i))
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			partFile := d.createPartFile(i)
			_, err = io.CopyBuffer(partFile, resp.Body, make([]byte, BufferSize))
			if err != nil {
				log.Fatal(err)
			}
			partFile.Close()
		}(i)
	}

	// Wait goroutines to finish
	d.WG.Wait()

	// Combine all part files into original file
	d.combinePartFiles()
}

func (d *downloader) getRange(presChunk int) string {
	chunkStart := presChunk * d.ChunkSize
	chunkEnd := (presChunk+1)*d.ChunkSize - 1
	if presChunk+1 == d.ConcurrencyCount {
		chunkEnd = d.ContentLength - 1
	}
	rangeValue := fmt.Sprintf("%s=%d-%d", d.AcceptRanges, chunkStart, chunkEnd)
	return rangeValue
}

func (d *downloader) setFileNameFromRawUrl() {
	Url, err := url.Parse(d.FileUrl)
	if err != nil {
		log.Fatal(err)
	}

	segments := strings.Split(Url.Path, "/")
	d.FileName = segments[len(segments)-1]
}

func (d *downloader) setHeadInfo() {
	head, err := http.Head(d.FileUrl)
	if err != nil {
		log.Fatal(err)
	}

	d.AcceptRanges = head.Header.Get(AcceptRanges)
	d.ContentLength, err = strconv.Atoi(head.Header.Get(ContentLength))
	if err != nil {
		d.ContentLength = -1
	}
}

func (d *downloader) setFileLocation() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	d.Location = filepath.Join(pwd, "Downloads")
}

func (d *downloader) createFile() {
	// create file in destination directory
	var err error
	d.File, err = os.Create(filepath.Join(d.Location, d.FileName))
	if err != nil {
		log.Fatal(err)
	}
}

func (d *downloader) createPartFile(partNo int) *os.File {
	// create part file in destination directory
	partFileName := fmt.Sprintf("%s.%d.part", d.FileName, partNo)
	partFile, err := os.Create(filepath.Join(d.Location, partFileName))
	if err != nil {
		log.Fatal(err)
	}
	return partFile
}

func (d *downloader) combinePartFiles() {
	for i := 0; i < d.ConcurrencyCount; i++ {
		partFileName := fmt.Sprintf("%s.%d.part", d.FileName, i)
		partFilePath := filepath.Join(d.Location, partFileName)
		partFile, err := os.Open(partFilePath)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.CopyBuffer(d.File, partFile, make([]byte, BufferSize))
		if err != nil {
			log.Fatal(err)
		}
		os.Remove(partFilePath)
	}
}

func (d *downloader) validate() {
	// FileName check
	if d.FileName == "" {
		d.setFileNameFromRawUrl()
	}

	// File Save location check
	if d.Location == "" {
		d.setFileLocation()
	}

	// Set info from head
	d.setHeadInfo()

	// concurrent downloader check
	if d.AcceptRanges != "" && d.ContentLength > 0 {
		if d.ConcurrencyCount < 1 {
			d.ConcurrencyCount = DefaultConcurrencyCount
		}
		if d.ConcurrencyCount > MaxConcurrencyCount {
			d.ConcurrencyCount = MaxConcurrencyCount
		}
		d.ChunkSize = d.ContentLength / d.ConcurrencyCount
	} else {
		d.ConcurrencyCount = DefaultConcurrencyCount
		d.ChunkSize = -1 // means that the file can't be downloaded in chunk
	}
}
