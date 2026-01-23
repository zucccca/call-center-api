package main

import (
	"fmt"
	"net/http"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hit upload handler")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10 << 20) // limit request to 10mb
	err := r.ParseMultipartForm( 10 << 20)

	if err != nil {
		http.Error(w, "File too big", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("audio")

	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}

	defer file.Close()

	fmt.Printf("Uploaded FIle: %+v\n", header.Filename)
	fmt.Printf("File Size: %+v\n", header.Size)

	fmt.Fprintf(w, "Successfully Uploaded File\n")

}