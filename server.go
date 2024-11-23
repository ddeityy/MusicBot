package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func RunServer() {
	http.HandleFunc("/", handleCookies)
	log.Println("Server listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}

func handleCookies(w http.ResponseWriter, r *http.Request) {
	log.Println("Got cookies request")

	cookie, err := r.Cookie("bruh")
	if err != nil {
		log.Println("Error getting cookie: ", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if cookie.Value != "moment" {
		log.Println("Wrong cookie value: ", "value", cookie.Value)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := os.Remove("./cookies.txt"); err != nil {
		log.Println("Error deleting old cookies.txt: ", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	log.Println("Deleted old cookies.txt")

	f, err := os.OpenFile("./cookies.txt", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Println("Error creating new cookies.txt: ", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()
	log.Println("Created new cookies.txt")

	_, err = io.Copy(f, r.Body)
	if err != nil {
		log.Println("Error copying cookies: ", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("Cookies written to cookies.txt")
}
