package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

func main() {
	http.HandleFunc("/", productListHandler)
	log.Fatal(http.ListenAndServe(":3001", nil))
}

func productListHandler(w http.ResponseWriter, r *http.Request) {
	products := []Product{
		{ID: 1, Name: "Product 1", Price: 100},
		{ID: 2, Name: "Product 2", Price: 200},
		{ID: 3, Name: "Product 3", Price: 300},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}
