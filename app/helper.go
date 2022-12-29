package app

import (
	"log"
	"runtime"
)

func memUsageCheck() {
	r := runtime.MemStats{}
	log.Println("Alloc =", r.Alloc)
	log.Println("Total Alloc =", r.TotalAlloc)
	log.Println("Sys =", r.Sys)
	log.Println("NumGC =", r.NumGC)
}
