package sharexhandler

import (
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"log"
	"fmt"
)

// Path configuration:
// All paths have to start with a slash ("/")
type PathConfiguration struct {
	UploadPath string // Path where POST-Requests of ShareX are routing at. Example: /upload
	GetPath    string // Path where clients get their files. The Id in the path must be {id}. Example: /get/{id}
}

// This is the main class which is used to use the ShareX handler
type ShareXHandler struct {
	// The path configuration
	PathConfiguration *PathConfiguration
	// The Storage where files will be stored at/loaded from
	Storage Storage
	// A function which is called on every request (for example to set specific response headers).
	OutgoingFunction func(http.ResponseWriter, *http.Request)
	// Buffer size in bytes which is allocated when sending a file.
	SendBufferSize int
	// Buffer size in bytes which is allocated when receiving a file.
	ReceiveBufferSize int
	// The path has to start a slash ("/"). This is where the router gets bound on.
	Path string
	// This is used to respond to upload requests and refer the ShareX client to the right url. It has to end with a slash! Example: http://localhost:8080/
	ProtocolHost string
	// Whitelisted content types which will be displayed in the client`s browser.
	WhitelistedContentTypes []string
	// Internal router to get the NotFoundHandler
	router *mux.Router
}

// This is the function which binds a ShareX handler router to the given path.
func (shareXHandler *ShareXHandler) BindToRouter(parentRouter *mux.Router) (subRouter *mux.Router) {
	router := parentRouter.PathPrefix(shareXHandler.Path).Subrouter()
	router.HandleFunc(shareXHandler.PathConfiguration.UploadPath, shareXHandler.handleUploadRequest)
	router.HandleFunc(shareXHandler.PathConfiguration.GetPath, shareXHandler.handleGetRequest)
	shareXHandler.router = router
	return router
}

func (shareXHandler *ShareXHandler) handleUploadRequest(w http.ResponseWriter, req *http.Request) {
	if shareXHandler.OutgoingFunction != nil {
		shareXHandler.OutgoingFunction(w, req)
	}
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
		log.Printf("An error occurred while parsing multipart form: %v\n", err)
	} else {
		if file, multipartFileHeader, err := req.FormFile("file"); err != nil {
			http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
			log.Printf("An error occurred while reading the form file: %v\n", err)
		} else {
			fileHeader := make([]byte, 512)
			_, err := file.Read(fileHeader)
			if err != nil {
				http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
				log.Printf("An error occurred while reading the file header: %v\n", err)
			}
			if _, err := file.Seek(0, 0); err != nil {
				panic(err)
			}
			fileName := multipartFileHeader.Filename
			mimeType := http.DetectContentType(fileHeader)
			entry := shareXHandler.Storage.NewStorageEntry()
			if err := entry.Save(); err != nil {
				http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
				log.Printf("An error occurred while saving a ShareX entry: %v\n", err)
			} else {
				id := entry.GetId()
				if writer, err := entry.GetWriter(); err != nil {
					http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
					log.Printf("An error occurred while getting the writer of the entry (%v): %v\n", entry.GetId(), err)
				} else {
					defer writer.Close()
					entry.SetContentType(mimeType)
					entry.SetFilename(fileName)
					var total int64 = 0
					for {
						buffer := make([]byte, shareXHandler.ReceiveBufferSize)
						bytesRead, err := file.Read(buffer)
						total += int64(bytesRead)
						if bytesRead == 0 {
							break
						} else if err != nil {
							http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
							log.Printf("An error occurred while buffering a piece of the body: %v\n", err)
						} else {
							writer.Write(buffer[:bytesRead])
						}
					}
					if err := entry.Update(); err != nil {
						http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
						log.Printf("An error occurred while updating the entry (%v): %v\n", entry.GetId(), err)
					} else {
						log.Printf("Created entry %v (%v bytes)\n", entry.GetId(), total)
						w.WriteHeader(http.StatusOK)
						dotIndex := strings.LastIndex(entry.GetFilename(), ".")
						var fileEnding string
						if dotIndex == -1 {
							fileEnding = ""
						} else {
							fileEnding = entry.GetFilename()[strings.LastIndex(entry.GetFilename(), "."):]
						}
						url := shareXHandler.ProtocolHost + id + fileEnding
						w.Write([]byte(url))
					}
				}
			}
		}
	}
}

var dispositionValueFormat = "%v; filename=\"%v\""

// This method handles get requests and shares files.
func (shareXHandler *ShareXHandler) handleGetRequest(w http.ResponseWriter, req *http.Request) {
	if shareXHandler.OutgoingFunction != nil {
		shareXHandler.OutgoingFunction(w, req)
	}
	vars := mux.Vars(req)
	id := vars["id"]
	lastDotIndex := strings.LastIndex(id, ".")
	if lastDotIndex == -1 {
		lastDotIndex = len(id)
	}
	id = id[:lastDotIndex]
	if success, err, entry := shareXHandler.Storage.LoadStorageEntry(id); err != nil {
		http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
		log.Printf("There was a database error while loading the entry %v: %v", entry.GetId(), err)
	} else if !success {
		shareXHandler.router.NotFoundHandler.ServeHTTP(w, req)
	} else if readSeeker, err := entry.GetReadSeeker(); err != nil {
		http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
		log.Printf("There was an error while reading the entry %v: %v", entry.GetId(), err)
	} else {
		// content-disposition: inline; filename="gogland_2017-07-10_18-29-32.png"
		// content-disposition: attachment; filename="temp.html"
		for _, value := range shareXHandler.WhitelistedContentTypes {
			if strings.EqualFold(value, entry.GetContentType()) {
				w.Header().Set("Content-Disposition", fmt.Sprintf(dispositionValueFormat, "inline", entry.GetFilename()))
				goto inlinePassed
			}
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(dispositionValueFormat, "attachment", entry.GetFilename()))
		w.Header().Set("Content-Type", entry.GetContentType())
	inlinePassed:
		http.ServeContent(w, req, "", entry.GetLastModifiedValue(), readSeeker)
	}
}
