package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func InitializeRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").Name("Index").HandlerFunc(controllers.getLock)
	return router
}

func main() {
	router := InitializeRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}
