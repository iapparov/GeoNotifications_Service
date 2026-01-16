package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Println("WEBHOOK RECEIVED:")
		log.Println(string(body))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Println("Webhook stub listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

//ngrok http 9090
