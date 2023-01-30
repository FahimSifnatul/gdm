package app

import (
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	symbol  string = "█"
	message string = "downloading..."
)

type ProgressBar interface {
	io.Writer

	// private methods
	resetRow()
	resetColor()
	getSpinner() string
	getElapsedTimeStr() string
	buildProgress()
	validate()
}

type progressBar struct {
	ContentLength int
	WrittenLength int
	Progress      []string
	SpinnerIndex  int // only if server has no concurrency support i.e. no ContentLength in url header

	Message   string
	Symbol    string
	RowNumber int
	Mutex     *sync.Mutex
	StartTime time.Time
}

// ensures that all methods are implemented
var _ ProgressBar = (*progressBar)(nil)

func NewProgressBar(contentLength, rowNumber int, mutex *sync.Mutex) *progressBar {
	pb := &progressBar{
		ContentLength: contentLength,
		Message:       message,
		Symbol:        symbol,
		RowNumber:     rowNumber,
		Mutex:         mutex,
		StartTime:     time.Now(),
	}
	pb.validate()
	return pb
}

func (pb *progressBar) Write(p []byte) (int, error) {
	pb.Mutex.Lock()
	defer pb.Mutex.Unlock()

	pb.resetRow()
	pb.resetColor()

	pLen := len(p)

	if pb.ContentLength == -1 {
		fmt.Printf("%s %s %s", pb.getSpinner(), pb.Message, pb.getElapsedTimeStr())
		pb.WrittenLength += pLen
		return pLen, nil
	}

	barPos := int(math.Round(float64(pLen+pb.WrittenLength) / float64(pb.ContentLength) * 100.00))
	if barPos < 0 {
		barPos = 0
	}
	if barPos > 100 {
		barPos = 0
	}
	fmt.Printf("%s %s", pb.Progress[barPos], pb.getElapsedTimeStr())
	pb.WrittenLength += pLen
	return pLen, nil
}

func (pb *progressBar) resetRow() {
	escapeStr := fmt.Sprintf("%s%d;1H", escape, pb.RowNumber)
	clearStr := fmt.Sprintf("%s%s", escape, clear)
	fmt.Printf("%s%s", escapeStr, clearStr)
}

func (pb *progressBar) resetColor() {
	c := escape + color
	fmt.Printf("%s", c)
}

func (pb *progressBar) getSpinner() string {
	currentSpinner := spinner[pb.SpinnerIndex]
	pb.SpinnerIndex++
	if pb.SpinnerIndex == len(spinner) {
		pb.SpinnerIndex = 0
	}
	return currentSpinner
}

func (pb *progressBar) getElapsedTimeStr() string {
	currentTime := time.Now()
	elapsedTime := currentTime.Sub(pb.StartTime)
	hour := int(elapsedTime.Hours())
	min := int(elapsedTime.Minutes()) % 60
	sec := int(elapsedTime.Seconds()) % 60
	elapsedTimeStr := fmt.Sprintf("[%02d:%02d:%02d]", hour, min, sec)
	return elapsedTimeStr
}

func (pb *progressBar) buildProgress() {
	// ContentLength check
	if pb.ContentLength == -1 {
		return
	}

	// build basic Progress string slice
	var baseProgress []string
	baseProgress = append(baseProgress, pb.Message+"|")
	for i := 1; i < 51; i++ {
		baseProgress = append(baseProgress, " ")
	}
	baseProgress = append(baseProgress, "|", "  0%")

	pb.Progress = append(pb.Progress, strings.Join(baseProgress, ""))

	for i := 1; i <= 100; i++ {
		// for 2% download completion, 1 symbol will be printed to terminal
		if i%2 == 0 {
			baseProgress[i/2] = pb.Symbol
		}
		// change percentage to current percentage
		baseProgress[len(baseProgress)-1] = fmt.Sprintf("%3d%%", i)
		pb.Progress = append(pb.Progress, strings.Join(baseProgress, ""))
	}
}

func (pb *progressBar) validate() {
	if pb.ContentLength <= 0 {
		pb.ContentLength = -1
	}
	pb.buildProgress()
}
