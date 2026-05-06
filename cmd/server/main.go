package main

import (
	"log" // 终端日志
	"net/http"
	"os_sp26_proj1/internal/api"
)

func main() {
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	addr := ":8080"
	log.Printf("server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
