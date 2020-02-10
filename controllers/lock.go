package controllers

import (
	"fmt"
	"net/http"
)

func getLock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	r.ParseForm()
	test := r.Form.Get("test")
	fmt.Println(test)

	return true
}
