package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"net/http"
	"time"
)
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/user/all", GetAllUserHandler).Methods("GET")
	r.HandleFunc("/user", GetUserDetailHandler).Methods("GET")
	r.HandleFunc("/user/set_password", SetUserPasswordHandler).Methods("PUT")
	r.HandleFunc("/user/set_mobile_number", SetMobileNumberHandler).Methods("PUT")
	r.HandleFunc("/user/search", SearchUserHandler).Methods("GET")
	r.HandleFunc("/login", Authentication).Methods("POST")
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"Content-Type","Authorization"},
		AllowedMethods: []string{"GET","POST","PUT","DELETE"},
	}).Handler(r)
	srv := &http.Server{
		Handler:      handler,
		Addr:         ":5501",            //5500 - Production; 5501 -Development
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Println("--Server Started--")
	log.Fatal(srv.ListenAndServe())
}
