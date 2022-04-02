package main

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/subtle"
	"flag"
	"fmt"
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
	SubmitLocation       string
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

var _rewrite *string
var _user string
var _pass string

var FileIdLookup = make(map[int]FileFolder)

func AuthHandler(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Incoming request from " + r.RemoteAddr)
		if _user == "" && _pass == "" {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, isOk := r.BasicAuth()
		if isOk {
			userHash := sha256.Sum256([]byte(user))
			_userHash := sha256.Sum256([]byte(_user))
			pwHash := sha256.Sum256([]byte(pass))
			_pwHash := sha256.Sum256([]byte(_pass))
			if subtle.ConstantTimeCompare(userHash[:], _userHash[:]) == 1 && subtle.ConstantTimeCompare(pwHash[:], _pwHash[:]) == 1 {
				log.Println("Access granted to " + r.RemoteAddr)
				next.ServeHTTP(w, r)
				return
			}
		}
		log.Println("Access denied to " + r.RemoteAddr)
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

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
		log.Println("Serving file " + file + " to " + r.RemoteAddr)
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
	log.Println("Serving file " + file + " to " + r.RemoteAddr)
	log.Println("Mime-Type detected: " + mime)
	w.Header().Add("Content-Type", mime)
	w.Header().Add("Content-Disposition", "attachment; filename="+strings.Split(r.RequestURI, "/")[len(strings.Split(r.RequestURI, "/"))-1])
	io.Copy(w, f)
}

func selection(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Disposition", "attachment; filename=ComposedDownload.zip")
	w.Header().Add("Content-Type", "application/zip")
	archive := zip.NewWriter(w)
	log.Println("Serving selection to " + r.RemoteAddr)
	r.ParseForm()
	for idx := range r.Form {
		i, _ := strconv.Atoi(idx)
		log.Println("Adding " + FileIdLookup[i].Path)
		AddFileToZip(FileIdLookup[i], archive)
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
			Icon:     *_rewrite + "/_html_/folder.png",
			LinkName: "../",
			Link:     *_rewrite + GetRelativePath(GetPreviousDirectory(CurrentDir)),
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
				Icon:     *_rewrite + "/_html_/folder.png",
				LinkName: entry.Name(),
				Link:     *_rewrite + GetRelativePath(CurrentDir) + entry.Name(),
			})
		} else {
			FileIdLookup[id] = FileFolder{
				Path:        CurrentDir + entry.Name(),
				isDirectory: false,
			}
			data = append(data, DirectoryListEntry{
				Id:       id,
				Icon:     *_rewrite + "/_html_/file.png",
				LinkName: entry.Name(),
				Link:     *_rewrite + "/files" + GetRelativePath(CurrentDir) + entry.Name(),
			})

		}
		id += 1
	}
	return Page{
		Title:                "Test",
		SubmitLocation:       *_rewrite + "/files/selection/",
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
	_port := flag.Int("port", 5000, "The port to listen on")
	_rewrite = flag.String("rewrite", "", "Append the given string to any URL response (for use with reverse proxies)")
	_user_ := flag.String("user", "", "The username to be used for auth")
	_pass_ := flag.String("pass", "", "The password to be used for auth")
	flag.Parse()
	_user = *_user_
	_pass = *_pass_
	RootDir = strings.ReplaceAll(*_flag, "\\", "/")
	CurrentDir = RootDir
	log.Println("Current directory is " + CurrentDir)
	log.Println(fmt.Sprintf("Listening on port %d", *_port))
	http.HandleFunc("/", AuthHandler(root))
	http.HandleFunc("/files/", AuthHandler(files))
	http.HandleFunc("/files/selection/", AuthHandler(selection))
	http.HandleFunc("/_html_/", AuthHandler(htmlfiles))
	http.ListenAndServe(fmt.Sprintf(":%d", *_port), nil)
}
