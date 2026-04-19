package main

import (
    "log"
    "net/http"
	
    "github.com/prestonty/notification-service/internal/config"
    "fmt"
)

func main() {
    cfg := config.Load()

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })

    addr := fmt.Sprintf(":%d", cfg.Port)
    log.Printf("server starting on %s", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}
