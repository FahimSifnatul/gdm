package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gdm/model"
)

const (
	AcceptRanges            string = "Accept-Ranges"
	ContentLength           string = "Content-Length"
	DefaultConcurrencyCount int    = 1
	MaxConcurrencyCount     int    = 16
	BufferSize              int    = 1048576 //bytes
)

type Downloader interface {
	NewDownloader()

	// private method
	getRange(presChunk int) (int, int)

	setFileNameFromRawUrl()
	setHeadInfo()
	setFileLocation()

	createFile()
	createPartFile(partNo int) *os.File
	combinePartFiles()
	getRowNumber()
	resetRow()

	validate()
}

type downloader struct {
	FileUrl  string
	Location string
	FileName string

	ConcurrencyCount int
	AcceptRanges     string
	ContentLength    int
	ChunkSize        int

	File  *os.File
	WG    sync.WaitGroup
	Mutex sync.Mutex
	Row   int // from where progress bars will start to print
}

// ensures that all methods are implemented
var _ Downloader = (*downloader)(nil)

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

	// reset terminal cursor to the next line of last progress bar
	d.resetRow()
}

func (d *downloader) NewDownloader() {
	// Define goroutines numbers to wait in WaitGroup
	d.WG.Add(d.ConcurrencyCount)

	// concurrently download the file
	for i := 1; i <= d.ConcurrencyCount; i++ {
		go func(i int) {
			defer d.WG.Done()
			client := http.Client{}

			req, err := http.NewRequest(http.MethodGet, d.FileUrl, nil)
			if err != nil {
				log.Fatal(err)
			}

			partFileLength := -1
			if d.ChunkSize != -1 {
				start, end := d.getRange(i)
				rangeStr := fmt.Sprintf("%s=%d-%d", d.AcceptRanges, start, end)
				req.Header.Add("Range", rangeStr)
				partFileLength = end - start + 1
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			partFile := d.createPartFile(i)
			bar := NewProgressBar(partFileLength, d.Row+i, &d.Mutex)
			_, err = io.CopyBuffer(io.MultiWriter(partFile, bar), resp.Body, make([]byte, BufferSize))
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

func (d *downloader) getRange(presChunk int) (int, int) {
	chunkStart := (presChunk - 1) * d.ChunkSize
	chunkEnd := presChunk*d.ChunkSize - 1
	if presChunk == d.ConcurrencyCount {
		chunkEnd = d.ContentLength - 1
	}
	return chunkStart, chunkEnd
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
	// create a part file in destination directory
	partFileName := fmt.Sprintf("%s.%d.part", d.FileName, partNo)
	partFile, err := os.Create(filepath.Join(d.Location, partFileName))
	if err != nil {
		log.Fatal(err)
	}
	return partFile
}

func (d *downloader) combinePartFiles() {
	for i := 1; i <= d.ConcurrencyCount; i++ {
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

func (d *downloader) getRowNumber() {
	// first print new lines equal to concurrency count
	// because if we don't print new lines then
	// if gdm command is executed at the last line of terminal
	// then progress bars won't show correctly
	for i := 1; i <= d.ConcurrencyCount; i++ {
		fmt.Printf("\n")
	}

	cmd := exec.Command("./app/row.sh")
	output, err := cmd.Output()
	if err != nil {
		log.Println(err)
	}
	r := strings.Split(string(output), "\n")[0]
	row, err := strconv.Atoi(r)
	if err != nil {
		log.Fatal("can't parse starting row number to print progress bar(s)")
	}
	d.Row = row - d.ConcurrencyCount
}

func (d *downloader) resetRow() {
	fmt.Printf("%s%d;1H", escape, d.Row+d.ConcurrencyCount+1)
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
		d.ChunkSize = -1 // means that file can't be downloaded in chunk
	}

	d.Mutex = sync.Mutex{}

	d.getRowNumber()
}
