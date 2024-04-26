package main

import (
	"fmt"
	"time"

	"net"
	"net/http"

	"golang.org/x/time/rate"
	"log"

	"context"
	"os"
	"os/signal"

	"runtime"
	"runtime/debug"
)

// System Settings.
const (
	filePath string = "public"

	host string = "127.0.0.1"
	port string = "8000"

	urlLength int = 1000

	frameCount int = 10
)

var (
	urlSetting []*urlRotator = []*urlRotator{
		{"/", world},
	}
)

var (
	limiter = rate.NewLimiter(1, 4) // 秒単位で４アクセス
)

//
type urlRotator struct {
	path     string
	function func(w http.ResponseWriter, r *http.Request)
}

//
func main() {
	//
	mux := http.NewServeMux()
	for _, f := range urlSetting {
		mux.HandleFunc("/exec"+f.path, f.function)
	}
	mux.HandleFunc("/", publicFile)

	urlSetting = nil // メモリから削除

	//
	server := &http.Server{
		Addr:    net.JoinHostPort(host, port),
		Handler: start(mux),

		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
	}

	runtime.GC()
	debug.SetGCPercent(-1)

	fmt.Println("start Shiba server.")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
	defer cancel()

	fmt.Println("shutdown.")
	server.Shutdown(ctx)

	debug.SetGCPercent(100)
}

var frameCounter int = 0

func start(next http.Handler) http.Handler {
	//
	frameCounter++
	if frameCounter == frameCount {
		frameCounter = 0

		rtsGarbageCollection()
	}

	//
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		if len(r.URL.Path) > urlLength {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func rtsGarbageCollection() {
	var mem runtime.MemStats

	runtime.ReadMemStats(&mem)

	if mem.HeapAlloc > 500<<20 { // 500MB
		runtime.GC()
	}
}

func publicFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filePath+r.URL.Path)
}

//

func world(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "New World.")
}
