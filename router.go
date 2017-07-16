package sharexhandler

import (
	"github.com/gorilla/mux"
	"net/http"
	"io"
	"mime"
	"mime/multipart"
	"bytes"
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
	// Buffer size in bytes which is allocated when sending a file. Per default this is set to 1024.
	BufferSize int
	// The path has to start a slash ("/"). This is where the router gets bound on.
	Path string
}

// This is the function which binds a ShareX handler router to the given path.
func (shareXHandler *ShareXHandler) BindToRouter(parentRouter *mux.Router) {
	router := parentRouter.PathPrefix(shareXHandler.Path).Subrouter()
	router.HandleFunc(shareXHandler.PathConfiguration.UploadPath, shareXHandler.handleUploadRequest)
	router.HandleFunc(shareXHandler.PathConfiguration.GetPath, shareXHandler.handleGetRequest)
}

// This method handles incoming POST upload request.
func (shareXHandler *ShareXHandler) handleUploadRequest(w http.ResponseWriter, req *http.Request) {
	if shareXHandler.OutgoingFunction != nil {
		shareXHandler.OutgoingFunction(w, req)
	}
	var err error
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, "400 bad request", http.StatusBadRequest)
	} else {
		multipartReader := multipart.NewReader(req.Body, params["boundary"])
		entry := shareXHandler.Storage.NewStorageEntry()
		if err := entry.Save(); err != nil {
			http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
			panic(err)
		} else {
			id := entry.GetId()
			if writer, err := entry.GetWriter(); err != nil {
				http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
				panic(err)
			} else {
				defer writer.Close()
				var partErr error
				var part *multipart.Part
				part, partErr = multipartReader.NextPart()
				if partErr != nil {
					http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
					panic(partErr)
				} else {
					buf := new(bytes.Buffer)
					entry.SetContentType(part.Header.Get("Content-Type"))
					entry.SetFilename(part.FileName())
					for ; ; {
						if partErr == nil {
							buf.Reset()
							if _, err := io.Copy(buf, part); err != nil {
								http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
								panic(err)
							} else {
								writer.Write(buf.Bytes())
							}
						} else if partErr != io.EOF {
							http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
							panic(partErr)
						} else {
							break
						}
						part, partErr = multipartReader.NextPart()
					}
					buf.Reset()
					if err := entry.Update(); err != nil {
						http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
						panic(partErr)
					} else {
						w.WriteHeader(200)
						w.Write([]byte("http://localhost:8080/sharex/"+id))
						//http.Redirect(w, req, shareXHandler.Path+shareXHandler.PathConfiguration.GetPath+id, http.StatusTemporaryRedirect)
					}
				}
			}
		}
	}
}

// This method handles get requests and shares files.
func (shareXHandler *ShareXHandler) handleGetRequest(w http.ResponseWriter, req *http.Request) {
	if shareXHandler.OutgoingFunction != nil {
		shareXHandler.OutgoingFunction(w, req)
	}
	vars := mux.Vars(req)
	id := vars["id"]
	if success, err, entry := shareXHandler.Storage.LoadStorageEntry(id); err != nil {
		http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
		panic(err)
	} else if !success {
		http.NotFound(w, req)
	} else {
		if reader, err := entry.GetReader(); err != nil {
			http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
			panic(err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", entry.GetContentType())
			buf := make([]byte, shareXHandler.BufferSize)
			for {
				n, err := reader.Read(buf)
				if err != nil && err != io.EOF {
					http.Error(w, "500 an internal error occurred", http.StatusInternalServerError)
					panic(err)
				}
				if n == 0 {
					break
				}
				if _, err := w.Write(buf[:n]); err != nil {
					panic(err)
				}
			}
		}
	}
}
