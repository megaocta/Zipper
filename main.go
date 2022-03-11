package main

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
)

func root(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Disposition", "attachment; filename=ComposedDownload.zip")
	w.Header().Add("Content-Type", "application/zip")
	archive := zip.NewWriter(w)
	AddFileToZip("file.msi", archive)
	AddFileToZip("file.pdf", archive)
	archive.Close()
}

func AddFileToZip(filename string, archive *zip.Writer) {
	writer, _ := archive.Create(filename)
	f, _ := os.Open(filename)
	io.Copy(writer, f)
}

func main() {

	http.HandleFunc("/", root)
	http.ListenAndServe(":5000", nil)
}
