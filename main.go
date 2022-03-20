package main

import (
	"archive/zip"
	"flag"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type FileFolder struct {
	Path        string
	isDirectory bool
}

type Page struct {
	Title                string
	DirectoryListEntries []DirectoryListEntry
}

type DirectoryListEntry struct {
	Id       int
	Icon     string
	Link     string
	LinkName string
}

// Path is a filesystem path
var CurrentDir string
var RootDir string
var ProgramDir string

var FileIdLookup = make(map[int]FileFolder)

func root(w http.ResponseWriter, r *http.Request) {
	CurrentDir = GetAbsolutePath(r.URL.Path)
	t := template.Must(template.ParseFiles("_html_/template.html"))
	err := t.Execute(w, ListDirectory())
	if err != nil {
		log.Fatal(err)
	}
}

func files(w http.ResponseWriter, r *http.Request) {
	data, _ := url.QueryUnescape(r.RequestURI)
	path := strings.Split(data, "/files/")
	if len(path) > 1 {
		f, err := os.Open(filepath.Join(RootDir, path[1]))
		if err != nil {
			w.Write([]byte("404 Not Found"))
		}
		file := filepath.Join(RootDir, path[1])
		mime := mime.TypeByExtension("." + strings.Split(file, ".")[len(strings.Split(file, "."))-1])
		if mime == "" {
			mime = "text/plain"
		}
		log.Println("Serving file " + file)
		log.Println("Mime-Type detected: " + mime)
		w.Header().Add("Content-Type", mime)
		w.Header().Add("Content-Disposition", "attachment; filename="+strings.Split(r.RequestURI, "/")[len(strings.Split(r.RequestURI, "/"))-1])
		io.Copy(w, f)
	} else {
		w.Write([]byte("404 Not Found"))
	}
}

func htmlfiles(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(filepath.Join(ProgramDir, r.RequestURI))
	if err != nil {
		w.Write([]byte("404 Not Found"))
	}
	file := filepath.Join(ProgramDir, r.RequestURI)
	mime := mime.TypeByExtension("." + strings.Split(file, ".")[len(strings.Split(file, "."))-1])
	if mime == "" {
		mime = "text/plain"
	}
	log.Println("Serving file " + file)
	log.Println("Mime-Type detected: " + mime)
	w.Header().Add("Content-Type", mime)
	w.Header().Add("Content-Disposition", "attachment; filename="+strings.Split(r.RequestURI, "/")[len(strings.Split(r.RequestURI, "/"))-1])
	io.Copy(w, f)
}

func selection(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Disposition", "attachment; filename=ComposedDownload.zip")
	w.Header().Add("Content-Type", "application/zip")
	archive := zip.NewWriter(w)
	r.ParseForm()
	for idx, _ := range r.Form {
		i, _ := strconv.Atoi(idx)
		AddFileToZip(FileIdLookup[i], archive)
		log.Println("Adding " + FileIdLookup[i].Path)
	}
	// AddFileToZip("file.msi", archive)
	// AddFileToZip("file.pdf", archive)
	archive.Close()
}

func ListDirectory() Page {
	id := 0
	entries, _ := os.ReadDir(CurrentDir)
	var data []DirectoryListEntry
	if CurrentDir != RootDir+"/" {
		data = append(data, DirectoryListEntry{
			Icon:     "/_html_/folder.png",
			LinkName: "../",
			Link:     GetRelativePath(GetPreviousDirectory(CurrentDir)),
		})
	}
	for _, entry := range entries {
		if entry.IsDir() {
			FileIdLookup[id] = FileFolder{
				Path:        CurrentDir + entry.Name(),
				isDirectory: true,
			}
			data = append(data, DirectoryListEntry{
				Id:       id,
				Icon:     "/_html_/folder.png",
				LinkName: entry.Name(),
				Link:     GetRelativePath(CurrentDir) + entry.Name(),
			})
		} else {
			FileIdLookup[id] = FileFolder{
				Path:        CurrentDir + entry.Name(),
				isDirectory: false,
			}
			data = append(data, DirectoryListEntry{
				Id:       id,
				Icon:     "/_html_/file.png",
				LinkName: entry.Name(),
				Link:     "/files" + GetRelativePath(CurrentDir) + entry.Name(),
			})

		}
		id += 1
	}
	return Page{
		Title:                "Test",
		DirectoryListEntries: data,
	}
}

func GetPreviousDirectory(path string) string {
	//s := strings.Split(path, "/")
	s := regexp.MustCompile("\\\\|/").Split(path, -1)
	s = s[0 : len(s)-2]
	var val string
	for i := 0; i < len(s); i++ {
		val += s[i] + "/"
	}
	return val
}

//Takes filesystem path returns web path
func GetAbsolutePath(relativePath string) string {
	val := RootDir + relativePath
	if val[len(val)-1:] != "/" && val[len(val)-1:] != "\\" {
		val += "/"
	}
	return val
}

// Takes web path returns filesystem path
func GetRelativePath(absolutePath string) string {
	return strings.Replace(absolutePath, RootDir, "", -1)
}

func ChangeSeparator(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func AddFileToZip(toAdd FileFolder, archive *zip.Writer) {
	if !toAdd.isDirectory {
		// to Add is a file
		writer, _ := archive.Create(filepath.Base(toAdd.Path))
		f, _ := os.Open(toAdd.Path)
		io.Copy(writer, f)
	} else {
		// to Add is a dir
		AddFolderToZip(filepath.Dir(toAdd.Path), filepath.Base(toAdd.Path), archive)
	}
}

func AddFolderToZip(ZipRoot string, relativeRoot string, archive *zip.Writer) {
	// Created the passed directory
	//writer, _ := archive.Create(relativeRoot)
	// Start walking through the passed directory
	ZipRoot = ChangeSeparator(ZipRoot)
	relativeRoot = ChangeSeparator(relativeRoot)
	filepath.Walk(filepath.Join(ZipRoot, relativeRoot), func(path string, i os.FileInfo, err error) error {
		if !i.IsDir() {
			// If a file was found
			writer, _ := archive.Create(strings.Replace(ChangeSeparator(path), ZipRoot+"/", "", 1))
			f, _ := os.Open(path)
			io.Copy(writer, f)
		}
		return nil
	})
}

func main() {
	ProgramDir, _ = os.Getwd()
	ProgramDir = strings.ReplaceAll(ProgramDir, "\\", "/")
	_flag := flag.String("root", ProgramDir, "The root directory for the webserver")
	flag.Parse()
	RootDir = strings.ReplaceAll(*_flag, "\\", "/")
	CurrentDir = RootDir
	log.Println("Current directory is " + CurrentDir)
	http.HandleFunc("/", root)
	http.HandleFunc("/files/", files)
	http.HandleFunc("/files/selection/", selection)
	http.HandleFunc("/_html_/", htmlfiles)
	http.ListenAndServe(":5000", nil)
}
