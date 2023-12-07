/* Server structures from my own SCION_TV Project (https://github.com/Nils-Treuheit/SCION-TV) 
   based on Marten Gartner's work in scion-apps (https://github.com/netsec-ethz/scion-apps/tree/master/_examples/shttp) */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"path/filepath"
	"strings"
	"crypto/tls"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/handlers"
	"github.com/netsec-ethz/scion-apps/pkg/shttp"
)

// relatively simple webserver fileserver topology
// derived from the SCION-TV project
func main() 
{
	local := flag.String("local", "0.0.0.0:8899", "The local HTTP or SCION address on which the server will be listening")
	remote := flag.String("remote", "19-ffaa:1:bcc,[127.0.0.1]:9988", "The SCION or HTTP address on which the server will be requested")
	webServPort := flag.String("webServPort", "80", "The port to serve website on")
	tslServPort := flag.String("tslServPort", "433", "The tsl port to serve website with tsl certificate on")
	webDir := flag.String("webDir", "html", "The directory of static webpage elements to host")
	fileServPort := flag.String("fileServPort", "8080", "The port to serve website on")
	fileDir := flag.String("fileDir", "hls", "The directory of streaming content to host")
	flag.Parse()

	go file_server(fileDir,fileServPort)
	go web_server(webDir,tslServPort,webServPort)
	proxy(local,remote)
}

// addHeaders will act as middleware to give us CORS support
func addHeaders(h http.Handler) http.HandlerFunc 
{
	return func(w http.ResponseWriter, r *http.Request) 
	{
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

/*
This is a very simple static file server in go
Navigating to http://localhost:8080 will display the directory file listings.
*/
func file_server(directory *string, port *string) 
{
	// Sample video from https://www.youtube.com/watch?v=xj2heO4-u-8
	mux := http.NewServeMux()
	mux.Handle("/", addHeaders(http.FileServer(http.Dir(*directory))))

	log.Printf("File-Server serves %s folder's streaming content on HTTP port: %s\n", *directory, *port )
	log.Fatalf("%s", http.ListenAndServe(":"+*port, mux))
}

/*
This is a very simple webpage server in go
Navigating to https://localhost:433 or http://localhost:80 will display the index.html.
*/
func web_server(webDir *string, tslPort *string, webPort *string)
{
	webpage := "index.html"
	website := *webDir+"/"+webpage
	icon := *webDir+"/favicon.ico"
	pic := *webDir+"/background.png"

	certFile := flag.String("cert", "", "Path to TLS server certificate for optional https")
	keyFile := flag.String("key", "", "Path to TLS server key for optional https")
	flag.Parse()

	m := http.NewServeMux()

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){http.ServeFile(w, r, website)})
	m.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request){http.ServeFile(w, r, icon)})
	m.HandleFunc("/background.png", func(w http.ResponseWriter, r *http.Request){http.ServeFile(w, r, pic)})


	// handler that responds with a friendly greeting
	m.HandleFunc("/hello-world", func(w http.ResponseWriter, r *http.Request) 
	  {
		// Status 200 OK will be set implicitly
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`Hello World! I am a dummy website to test the impact of SCION path selection on the performance of content delivery.`))
	  }
	)

	// handler that responds with an image file
	m.HandleFunc("/sample-image", func(w http.ResponseWriter, r *http.Request) 
	  {
		// serve the sample JPG file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"SCION.JPG") 
		// Sample image from https://blog.apnic.net/wp-content/uploads/2021/09/SCION-FT-555x202.jpg?v=1670e3759db62840c91aa22608946e73
	  }
	)
	
	// handler that responds with an gif file
	m.HandleFunc("/sample-gif", func(w http.ResponseWriter, r *http.Request) 
	  {
		// serve the sample GIF file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"boycott.gif") 
		// Sample image from https://giphy.com/gifs/southparkgifs-3o6ZsZTFpJxmZaeqcw
	  }
	)

	// handler that responds with an audio file
	m.HandleFunc("/sample-audio", func(w http.ResponseWriter, r *http.Request) 
	  {
		// serve the sample AUDIO file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"Chopin-nocturne-op-9-no-2.mp3") 
		// Sample music from https://orangefreesounds.com/chopin-nocturne-op-9-no-2
	  }
	)

	// handler that responds with an video file
	m.HandleFunc("/sample-video", func(w http.ResponseWriter, r *http.Request) 
	  {
		// serve the sample MP4 file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"SCION_DDoS_Def.mp4") 
		// Sample video from https://www.youtube.com/watch?v=-JeEppbCZTw
	  }
	)
	

	// GET handler that responds with some json data
	m.HandleFunc("/sample-json", func(w http.ResponseWriter, r *http.Request) 
	  {
		if r.Method == http.MethodGet 
		{
			data := struct {
				Time    string
				Agent   string
				Proto   string
				Message string
			}{
				Time:    time.Now().Format("2006.01.02 15:04:05"),
				Agent:   r.UserAgent(),
				Proto:   r.Proto,
				Message: "success",
			}
			resp, _ := json.Marshal(data)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, string(resp))
		} 
		else { http.Error(w, "wrong method: "+r.Method, http.StatusForbidden) }
	  }
	)

	// POST handler that responds by parsing form values and returns them as string
	m.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) 
	  {
		if r.Method == http.MethodPost 
		{
			if err := r.ParseForm(); err != nil 
			{
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "received following data:\n")
			for s := range r.PostForm 
			{ fmt.Fprint(w, s, "=", r.PostFormValue(s), "\n") }
		} 
		else { http.Error(w, "wrong method: "+r.Method, http.StatusForbidden) }
	  }
	)

	handler := handlers.LoggingHandler(os.Stdout, m)
	if *certFile != "" && *keyFile != "" 
	{
		go func() { log.Fatal(shttp.ListenAndServeTLS(":"+*tslPort, *certFile, *keyFile, handler)) }()
		log.Printf("Web-Server serves %s webpage and its elements on HTTP port: %s\n", webpage, *tslPort)
	}
	else { log.Printf("Web-Server serves %s webpage and its elements on HTTP port: %s\n", webpage, *webPort) }
	log.Fatal(shttp.ListenAndServe(":"+*webPort, handler))
}


/*
PLEASE NOTE:
------------
This proxy implementation is pretty much a copy of the proxy implementation from the official scion-apps repository[https://github.com/netsec-ethz/scion-apps]
The current implementation of the official proxy can be found here[https://github.com/netsec-ethz/scion-apps/blob/master/_examples/shttp/proxy/main.go].
There will be differences between mine and the linked implementation since the repositories are not synced.

This proxy is used as a SCION to IP bridge in order to make a broadcasted Web-Content available to the local machine/network.
The proxy will be used as a SCION ingress proxy.

Navigating to http://localhost:8899 to access the streamed content.
*/
func proxy(local *string, remote *string) {

	mux := http.NewServeMux()

	// parseUDPAddr validates if the address is a SCION address
	// which we can use to proxy to SCION
	if _, err := snet.ParseUDPAddr(*remote); err == nil {
		proxyHandler, err := shttp.NewSingleSCIONHostReverseProxy(*remote, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h3"}})
		if err != nil {
			log.Fatalf("Failed to create SCION reverse proxy %s", err)
		}

		mux.Handle("/", proxyHandler)
		log.Printf("Proxy connected to SCION remote %s\n", *remote)
	} else {
		u, err := url.Parse(*remote)
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed parse remote %s, %s", *remote, err))
		}
		log.Printf("Proxy connected to HTTP remote %s\n", *remote)
		mux.Handle("/", httputil.NewSingleHostReverseProxy(u))
	}

	if lAddr, err := snet.ParseUDPAddr(*local); err == nil {
		log.Printf("Proxy listens on SCION %s\n", *local)
		// ListenAndServe does not support listening on a complete SCION Address,
		// Consequently, we only use the port (as seen in the server example)
		log.Fatalf("%s", shttp.ListenAndServe(fmt.Sprintf(":%d", lAddr.Host.Port), mux, nil))
	} else {
		log.Printf("Proxy listens on HTTP %s\n", *local)
		log.Fatalf("%s", http.ListenAndServe(*local, mux))
	}
}