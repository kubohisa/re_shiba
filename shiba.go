package main

import (
	"fmt"
	"regexp"
	"strings"
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

// Server System Settings.

const (
	DEBUG bool = false
)

var (
	filePath string = "public"

	port string = "8000"

	urlLength int = 1000

	threadCount int = 10
) // サーバーのパラメーターの設定

var (
	urlSetting []*urlRotator = []*urlRotator{
		{"/", world},
	}
) // URLルーティングの設定

var (
	limiter = rate.NewLimiter(1, 4) // 秒単位で４アクセス
) // 実行スレッド数の設定

// Server.

type urlRotator struct {
	path     string
	function func(w http.ResponseWriter, r *http.Request)
}

var (
	rexUrl = regexp.MustCompile(`\A[A-Za-z0-9\%\#\$\-\_\.\+\!\*\'\(\)\,\;\/\?\:\@\=\&\~\\\|]+\z`)

	hostAddress string
)

func main() {
	//
	mux := http.NewServeMux()
	for _, f := range urlSetting {
		mux.HandleFunc("/exec"+strings.TrimSpace(f.path), f.function)
	}
	mux.HandleFunc("/health", healthCheck)
	mux.HandleFunc("/", publicFile)

	urlSetting = nil // メモリから削除

	//
	filePath = strings.TrimSpace(filePath)
	port = strings.TrimSpace(port)

	if port == "" {
		port = "80"
	}

	h := ""
	if DEBUG == true {
		h = "127.0.0.1"
	}

	a := net.JoinHostPort(h, port)

	server := &http.Server{
		Addr:    a,
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

var threadCounter int = 0

func start(next http.Handler) http.Handler {
	//
	threadCounter++
	if threadCounter == threadCount {
		threadCounter = 0

		rtsGarbageCollection()
	}

	//
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		waf(w, r)

		next.ServeHTTP(w, r)
	})
}

func waf(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) > urlLength {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	if rexUrl.FindString(r.URL.Path) == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}
}

func rtsGarbageCollection() {
	var mem runtime.MemStats

	runtime.ReadMemStats(&mem)

	if mem.HeapAlloc > 500<<20 { // 500MB
		runtime.GC()
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(200), http.StatusOK)
	return
}

func publicFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filePath+r.URL.Path)
}

// Action(s).

func world(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "New World.")
}
