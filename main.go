package main

import (
	"log"
	"net/http"
)

func main() {
	db := InitDB()
	defer db.Close()

	server := NewServer(db)

	http.HandleFunc("/products", server.CreateProduct)
	http.HandleFunc("/orders", server.PlaceOrder)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
