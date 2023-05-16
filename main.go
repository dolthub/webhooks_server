package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var port = flag.Int("port", 1709, "http listening port")

func main() {
	flag.Parse()

	if *port == 0 {
		fmt.Println("must supply --port")
		os.Exit(1)
	}

	httpSrv := getHttpServer(*port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		signal.Stop(quit)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		fmt.Println("http server is shutting down")
		if err := httpSrv.Shutdown(ctx); err != nil {
			fmt.Println("failed to shutdown http server", err.Error())
		}
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Serving http on :", *port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error serving http server:", err.Error())
		}
	}()

	wg.Wait()

}

func handleWebhookEvents(w http.ResponseWriter, req *http.Request) {
	bb, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println("failed to read request body:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	for k, v := range req.Header {
		fmt.Printf("request headers: %s: %v\n", k, v)
	}

	fmt.Println("request body:", string(bb))
	w.WriteHeader(http.StatusOK)
}

func getHttpServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusBadRequest)
			_, err := io.WriteString(writer, "only POST requests supported.")
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println("received unsupported request method")
		} else {
			handleWebhookEvents(writer, request)
		}
	})
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}
