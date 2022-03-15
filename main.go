package main

import (
	"archive/zip"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Page struct {
	Title                string
	DirectoryListEntries []DirectoryListEntry
}

type DirectoryListEntry struct {
	Icon string
	Link string
}

var CurrentDir string
var RootDir string

func root(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("template.html"))
	err := t.Execute(w, ListDirectory(CurrentDir))
	if err != nil {
		log.Fatal(err)
	}
	// w.Header().Add("Content-Disposition", "attachment; filename=ComposedDownload.zip")
	// w.Header().Add("Content-Type", "application/zip")
	// archive := zip.NewWriter(w)
	// AddFileToZip("file.msi", archive)
	// AddFileToZip("file.pdf", archive)
	// archive.Close()
}

func files(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.RequestURI, "/files/")
	if len(path) > 1 {
		f, err := os.Open(filepath.Join(CurrentDir, path[1]))
		if err != nil {
			w.Write([]byte("404 Not Found"))
		}
		io.Copy(w, f)
	} else {
		w.Write([]byte("404 Not Found"))
	}
}

func ListDirectory(path string) Page {
	entries, _ := os.ReadDir(CurrentDir)
	var data []DirectoryListEntry
	for _, entry := range entries {
		if entry.IsDir() {
			data = append(data, DirectoryListEntry{
				Icon: "files/folder.png",
				Link: entry.Name(),
			})
		} else {
			data = append(data, DirectoryListEntry{
				Icon: "files/file.png",
				Link: entry.Name(),
			})

		}
	}
	return Page{
		Title:                "Test",
		DirectoryListEntries: data,
	}
}

func AddFileToZip(filename string, archive *zip.Writer) {
	writer, _ := archive.Create(filename)
	f, _ := os.Open(filename)
	io.Copy(writer, f)
}

func main() {
	CurrentDir, _ = os.Getwd()
	log.Println("Current directory is " + CurrentDir)
	RootDir = CurrentDir
	http.HandleFunc("/", root)
	http.HandleFunc("/files/", files)
	http.ListenAndServe(":5000", nil)
}
