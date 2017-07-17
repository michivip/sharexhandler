# ShareX custom upload handler
This is a simple library which can be used to create your own custom Golang upload handler for your ShareX client. It is based on the [http mux library](https://github.com/gorilla/mux) which I really do recommend for working with http handlers in Golang.
## Introduction
In ShareX you can choose between a bunch of different default built-in upload servers - but according to my expirience it was nearly impossible to manage all my uploaded files. Thats why I decided to write my own upload server (library). I chose Golang because it is a young and really powerful programming language which in my opinion has much potencial.
## Tutorial
I will not write an own Wiki for this repository because in my opinion the usage is really simple and by explaining you some examples you should be able to use the code.
### Design
The complete code is based on interfaces which can be used to code an own implementation. Moreover you can customize all paths/uris which should be used to handle processes. If thats still not enough you can just edit the code directly because of the really flexible [MIT license](https://github.com/michivip/sharexhandler/blob/master/LICENSE).
### Examples
The tutorial for itself only contains a basic example which is explained with comments. The complete design code (interfaces, basic methods/structs) is commented so that you can find your way through the code.
#### Basic example with SSL and some custom headers
```Go
package main

import (
	"github.com/michivip/sharexhandler"
	"net/http"
	"github.com/gorilla/mux"
	"crypto/tls"
	"strings"
	"log"
	"os"
	"github.com/michivip/sharexhandler/storages"
	"gopkg.in/mgo.v2"
	"bufio"
)

func main() {
	log.Println("Initializing storage...")
	fileFolder := "files/" // The folder where our uploaded files will be stored at.
	if err := os.Mkdir(fileFolder, os.ModePerm); err != nil { // Creating the folder
		panic(err)
	} else {
		// This example goes with the default built-in implemented MongoStorage but in your case you can use a different one.
		storage := &storages.MongoStorage{
			DialInfo: &mgo.DialInfo{ // The dial info. More information: https://godoc.org/labix.org/v2/mgo#DialInfo
				Addrs: []string{"localhost"},
			},
			Configuration: &storages.MongoStorageConfiguration{ // MongoDB configuration
				DatabaseName:         "sharex",   // Database name where collections are created in.
				UploadCollectionName: "uploads",  // Collection where the upload file information (not the file data) is stored at.
				FileFolderPath:       fileFolder, // The folder where the file data is stored at - no information.
			},
		}
		if err := storage.Initialize(); err != nil { // Initializing the storage - in our case connecting to the database.
			panic(err)
		} else {
			log.Println("Initialized!")
			log.Println("Set up custom upload server. Running it in background...")
			srv := setupHttpsServer(storage)
			go srv.ListenAndServeTLS("cert.pem", "private.pem") // Our cert file and our key file which are laying in our directory.
			// Now just a stop hook - nothing special.
			log.Println("Done! Enter \"stop\" to stop the webservers.")
			reader := bufio.NewReader(os.Stdin)
			var text string
			for ; strings.TrimSuffix(text, "\n") != "stop"; text, _ = reader.ReadString('\n') {
			}
			log.Println("Stopping...")
			os.Exit(0)
		}
	}
}

// Main method where our SSL https server is set up in.
func setupHttpsServer(storage sharexhandler.Storage) *http.Server {
	// New router from the mux package
	router := mux.NewRouter()
	// Instantiating the main struct ShareXHandler.
	shareXHandler := &sharexhandler.ShareXHandler{
		// The path which has to start with a slash and should end with no slash. This path will be appended after the host. Example: https://localhost/share
		Path: "/share",
		// Configuration of all paths where requests are handled.
		PathConfiguration: &sharexhandler.PathConfiguration{
			// Upload POST requests are handled on this path. Example about: https://localhost/share/upload/
			UploadPath: "/upload/",
			// All GET requests which request uploaded files by their id with file ending.
			// The handler would be called with the example about with the following value: https://localhost/share/get/MYSSUPERCOOLID.PNG
			GetPath: "/get/{id}",
		},
		// The storage which is an implemented interface.
		Storage: storage,
		// Buffer size which is used to write data to the incoming GET request clients.
		BufferSize: 1024,
		// The host which is used to respond to an upload request and name the id URL.
		ProtocolHost: "https://localhost",
		// A custom handler hook to set headers.
		OutgoingFunction: handleTlsRequest,
		// This will display every png/jpg image and plain text (.txt file) in the client`s browser - every other content type will be downloaded manually.
		WhitelistedContentTypes: []string{"image/png", "image/jpg", "text/plain"},
	}
	// Internal method which binds the ShareX handler with the given configuration to the parent router.
	shareXHandler.BindToRouter(router)
	// A simple TLS configuration
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	// Listening on 443 to get incoming https requests.
	srv := &http.Server{
		Addr:         "localhost:443",
		Handler:      router,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	return srv
}

func handleTlsRequest(w http.ResponseWriter, req *http.Request) {
	// Some TLS headers.
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Header().Add("Content-Security-Policy", "default-src 'none'; font-src 'none'; img-src 'self'; object-src 'none'; script-src 'self'; style-src 'unsafe-inline'")
	w.Header().Add("X-Content-Type-Options", "nosniff")
	w.Header().Add("X-Frame-Options", "DENY")
	w.Header().Add("X-XSS-Protection", "1; mode=block")
	w.Header().Add("Server", "Golang Webserver")
}
```
## Used libraries
In this project I have used some different external libraries. The list below enumerates all of them:
* [mgo](https://github.com/go-mgo/mgo)
* [mux](https://github.com/gorilla/mux)
