package main

import (
	"fmt"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"time"

	"context"
	"os"
	"os/signal"

	"runtime"
	"runtime/debug"
)

//

var (
	filePath string = "public"

	urlSetting []*urlRotator = []*urlRotator{
		{"/", world},
	}

	limiter = rate.NewLimiter(1, 4) // 秒単位で４アクセス
)

type urlRotator struct {
	path     string
	function func(w http.ResponseWriter, r *http.Request)
}

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
		Addr:    "localhost:8000",
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
	ctx, cancel := context.WithTimeout(context.Background(), 14 * time.Second)
	defer cancel()

	fmt.Println("shutdown.")
	server.Shutdown(ctx)

	debug.SetGCPercent(100)
}

var frameCounter int = 0

func start(next http.Handler) http.Handler {
	// リアルタイム・ガベージコレクタ
	frameCounter++
	if frameCounter == 10 {
		frameCounter = 0

		var mem runtime.MemStats

		runtime.ReadMemStats(&mem)

		if mem.HeapAlloc > 500<<20 { // 500MB
			runtime.GC()
		}
	}

	//
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

        if (len(r.URL.Path) > 1000) {
            http.Error(w, http.StatusText(500), http.StatusInternalServerError)
            return			
		}

		next.ServeHTTP(w, r)
	})
}

func publicFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filePath+r.URL.Path)
}

//

func world(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "New World.")
}
