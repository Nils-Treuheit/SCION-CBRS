package main

/* Server structures from my own SCION_TV Project (https://github.com/Nils-Treuheit/SCION-TV)
   based on Marten Gartner's work in scion-apps (https://github.com/netsec-ethz/scion-apps/tree/master/_examples/shttp) */

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"

	"github.com/gorilla/handlers"
)

func main() {
	var ivrs pan.ReplySelector = NewSmartReplySelector(2)
	var vsrs pan.ReplySelector = NewSmartReplySelector(1)
	var grs pan.ReplySelector = NewSmartReplySelector(0)

	if len(os.Args) < 2 {
		fmt.Println("Execute smart round robin approach:")
		startServs(ivrs, vsrs, grs)
	} else {
		command := os.Args[1]

		switch command {
		case "nors":
			fmt.Println("Execute no reply selector approach:")
			ivrs = pan.NewDefaultReplySelector()
			vsrs = pan.NewDefaultReplySelector()
			grs = pan.NewDefaultReplySelector()
		case "rrrs":
			fmt.Println("Execute round robin reply selector approach:")
			ivrs = NewRRReplySelector()
			vsrs = NewRRReplySelector()
			grs = NewRRReplySelector()
		case "mturs":
			fmt.Println("Execute MTU filtered round robin approach:")
			ivrs = NewSmartReplySelector(0)
			vsrs = NewSmartReplySelector(0)
			grs = NewSmartReplySelector(0)
		case "latrs":
			fmt.Println("Execute latency filtered round robin approach:")
			ivrs = NewSmartReplySelector(1)
			vsrs = NewSmartReplySelector(1)
			grs = NewSmartReplySelector(1)
		case "bwrs":
			fmt.Println("Execute bandwidth filtered round robin approach:")
			ivrs = NewSmartReplySelector(2)
			vsrs = NewSmartReplySelector(2)
			grs = NewSmartReplySelector(2)
		default:
			fmt.Println("Your ReplySelector Strategy has not been implemented!")
			return
		}
		startServs(ivrs, vsrs, grs)
	}
	answer := 'c'
	time.Sleep(3 * time.Duration(time.Second))
	fmt.Println("Enter any character to terminate session...")
	fmt.Scanf("\n%c", answer)

}

// relatively simple webserver fileserver topology
// derived from the SCION-TV project
func startServs(ivrs pan.ReplySelector, vsrs pan.ReplySelector, grs pan.ReplySelector) {
	webServPort := flag.String("webServPort", "80", "The port to serve website on")
	contentServPort := flag.String("contentServPort", "8181", "The port to serve website-content on")
	tslServPort := flag.String("tslServPort", "433", "The tsl port to serve website with tsl certificate on")
	webDir := flag.String("webDir", "website", "The directory of static webpage elements to host")
	fileServPort := flag.String("fileServPort", "8899", "The port to serve website on")
	fileDir := flag.String("fileDir", "stream_files", "The directory of streaming content to host")
	flag.Parse()

	go file_server(fileDir, fileServPort, vsrs)
	go content_server(webDir, contentServPort, ivrs)
	go web_server(webDir, tslServPort, webServPort, grs)
}

// addHeaders will act as middleware to give us CORS support
func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

/*
This is a very simple static file server in go
Navigating to http://localhost:8899 will display the directory file listings.
*/
func file_server(directory *string, port *string, rs pan.ReplySelector) {
	// Sample video from https://www.youtube.com/watch?v=xj2heO4-u-8
	mux := http.NewServeMux()
	mux.Handle("/", addHeaders(http.FileServer(http.Dir(*directory))))

	log.Printf("File-Server serves %s folder's streaming content on HTTP port: %s\n", *directory, *port)
	log.Fatalf("%s", ListenAndServeRepSelect(":"+*port, mux, rs))
}

/*
This is a very simple webpage server in go
Navigating to https://localhost:8181 will display the index.html.
*/
func content_server(webDir *string, webPort *string, rs pan.ReplySelector) {
	pic := *webDir + "/background.png"

	m := http.NewServeMux()
	m.HandleFunc("/background.png", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, pic) })

	// handler that responds with a friendly greeting
	m.HandleFunc("/hello-world", func(w http.ResponseWriter, r *http.Request) {
		// Status 200 OK will be set implicitly
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`Hello World! I am a dummy website to test the impact of SCION path selection on the performance of content delivery.`))
	})

	// handler that responds with a chatGPT story
	m.HandleFunc("/sample-text", func(w http.ResponseWriter, r *http.Request) {
		// Status 200 OK will be set implicitly
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`Here a Chat GPT Story about the history of the Internet\n\n\nOnce upon a time, in the early days of the internet, connections were slow, and the digital world was a far cry from the intricate tapestry it is today. It was a time when the concept of "surfing the web" meant navigating through simple HTML pages, and the idea of streaming high-quality video or conducting seamless online transactions was but a distant dream.\n\nAs the years passed, a wave of advancements in internet technologies began to reshape the landscape. The first significant milestone was the advent of broadband connections, replacing the screeching sounds of dial-up modems with high-speed, always-on connectivity. This transition marked the beginning of a new era, enabling users to explore richer content and engage in more complex online activities.\n\nThe rise of Web 2.0 brought forth a paradigm shift. The internet ceased to be a static collection of pages and transformed into a dynamic platform for collaboration and interaction. Social media platforms emerged, connecting people across the globe in ways previously unimaginable. The internet was no longer just a source of information; it became a virtual social space where ideas, opinions, and cat videos could be shared instantaneously.\n\nSimultaneously, the field of e-commerce flourished. Secure online transactions became a reality, paving the way for the digital marketplace. Consumers could now browse, shop, and pay for goods and services without leaving the comfort of their homes. This not only revolutionized retail but also laid the foundation for a globalized economy where borders mattered less in the digital realm.\n\nThe internet's infrastructure also underwent significant improvements. Fiber-optic cables replaced traditional copper wiring, dramatically increasing data transfer speeds and bandwidth. The concept of cloud computing emerged, allowing users to store and access vast amounts of data remotely. This not only streamlined processes for individuals and businesses but also contributed to the rise of services such as online storage, streaming, and collaborative platforms.\n\nThe proliferation of mobile devices further fueled the internet's evolution. Smartphones became ubiquitous, bringing the internet into the palms of billions. Mobile applications transformed how we access information, communicate, and even manage our daily lives. The mobile revolution made the internet more accessible and personalized, blurring the lines between the digital and physical worlds.\n\nThe Internet of Things (IoT) emerged as yet another frontier. Everyday objects became "smart," equipped with sensors and connectivity, creating a vast network of interconnected devices. Homes, cities, and industries became more efficient and interconnected, laying the groundwork for a future where the internet seamlessly integrates into every aspect of our lives.\n\nAs we venture into the era of 5G connectivity, the internet's journey continues to unfold. Faster speeds, lower latency, and increased capacity promise a world where immersive experiences, such as augmented reality and virtual reality, become commonplace. The boundaries of what is possible on the internet continue to expand, with innovations like blockchain technology and artificial intelligence shaping its future.\n\nIn this ongoing saga of technological progress, the internet stands not just as a tool for information but as a dynamic force driving societal, economic, and cultural change. The journey is far from over, and as we navigate the ever-evolving landscape of internet technologies, the next chapter promises even more remarkable advancements and transformative possibilities.\n\nAmidst the evolving tapestry of internet technologies, a new chapter unfolds with the introduction of the Secure Internet Architecture, better known as SCION. In the quest for a safer and more efficient network, SCION emerges as a beacon of innovation, reshaping the very foundations of how data traverses the digital realm.\n\nSCION, born from the collective wisdom of internet pioneers, represents a paradigm shift in network architecture. Unlike traditional protocols, SCION abandons the one-size-fits-all approach in favor of a modular and flexible design. It introduces a novel concept of path-aware networking, allowing data packets to navigate through the most secure and efficient routes available.\n\nSecurity lies at the heart of SCION's transformative power. In a landscape where cyber threats loom large, SCION pioneers a departure from conventional security measures. Its inherent design mitigates risks associated with common vulnerabilities, such as route hijacking and man-in-the-middle attacks. By introducing a secure-by-design framework, SCION erects a formidable defense against the ever-evolving cyber threats that plague the internet.\n\nOne of the groundbreaking features of SCION is its ability to provide provable security guarantees. Traditional protocols often rely on assumptions, leaving room for uncertainties in the face of sophisticated attacks. SCION, on the other hand, employs a principled approach backed by mathematical proofs. This not only enhances the resilience of the network but instills confidence in users and organizations relying on its infrastructure.\n\nEfficiency is another frontier where SCION excels. Through its innovative path selection mechanism, SCION optimizes data transfer, minimizing latency and ensuring swift delivery. This efficiency is a game-changer for applications demanding real-time responsiveness, such as online gaming, video conferencing, and emerging technologies like the Internet of Things (IoT).\n\nAs SCION gains traction, it heralds a new era for the internet—a safer, more reliable, and efficient era. Industries, governments, and individuals alike recognize its potential to reshape digital landscapes, fortifying critical infrastructure and safeguarding sensitive information.\n\nGovernments find in SCION a trustworthy foundation for securing national communications, protecting against cyber threats that jeopardize national security. Businesses, eager to fortify their digital presence, embrace SCION as a catalyst for innovation, ensuring a resilient and secure environment for their operations.\n\nFor individuals navigating the digital realm, SCION becomes a guarantee of privacy and reliability. With the assurance that their data travels through the most secure channels, users can engage in online activities without the constant worry of unauthorized access or data breaches.\n\nAs SCION weaves its way into the fabric of the internet, its impact reverberates across continents and industries. The internet, once a realm of uncertainties, transforms into a secure and efficient network, fostering innovation, collaboration, and progress. SCION's journey is not just a technological evolution; it's a testament to the relentless pursuit of a safer and more interconnected world in the digital age. And so, the story of the internet's metamorphosis continues, guided by the transformative power of SCION.`))
	})

	// handler that responds with an image file
	m.HandleFunc("/sample-image", func(w http.ResponseWriter, r *http.Request) {
		// serve the sample JPG file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"/SCION.JPG")
		// Sample image from https://blog.apnic.net/wp-content/uploads/2021/09/SCION-FT-555x202.jpg?v=1670e3759db62840c91aa22608946e73
	})

	// handler that responds with an gif file
	m.HandleFunc("/sample-gif", func(w http.ResponseWriter, r *http.Request) {
		// serve the sample GIF file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"/boycott.gif")
		// Sample image from https://giphy.com/gifs/southparkgifs-3o6ZsZTFpJxmZaeqcw
	})

	// handler that responds with an audio file
	m.HandleFunc("/sample-audio", func(w http.ResponseWriter, r *http.Request) {
		// serve the sample AUDIO file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"/Chopin-nocturne-op-9-no-2.mp3")
		// Sample music from https://orangefreesounds.com/chopin-nocturne-op-9-no-2
	})

	// handler that responds with an video file
	m.HandleFunc("/sample-video", func(w http.ResponseWriter, r *http.Request) {
		// serve the sample MP4 file
		// Status 200 OK will be set implicitly
		// Content-Length will be inferred by server
		// Content-Type will be detected by server
		http.ServeFile(w, r, *webDir+"/SCION_DDoS_Def.mp4")
		// Sample video from https://www.youtube.com/watch?v=-JeEppbCZTw
	})

	// GET handler that responds with some json data
	m.HandleFunc("/sample-json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
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
		} else {
			http.Error(w, "wrong method: "+r.Method, http.StatusForbidden)
		}
	})

	// POST handler that responds by parsing form values and returns them as string
	m.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "received following data:\n")
			for s := range r.PostForm {
				fmt.Fprint(w, s, "=", r.PostFormValue(s), "\n")
			}
		} else {
			http.Error(w, "wrong method: "+r.Method, http.StatusForbidden)
		}
	})

	handler := handlers.LoggingHandler(os.Stdout, m)
	log.Printf("Content-Server serves webpage content on HTTP port: %s\n", *webPort)
	log.Fatalf("%s", ListenAndServeRepSelect(":"+*webPort, handler, rs))
}

/*
This is a very simple webpage server in go
Navigating to https://localhost:433 or http://localhost:80 will display the index.html.
*/
func web_server(webDir *string, tslPort *string, webPort *string, rs pan.ReplySelector) {
	webpage := "index.html"
	website := *webDir + "/" + webpage
	icon := *webDir + "/favicon.ico"

	certFile := flag.String("cert", "", "Path to TLS server certificate for optional https")
	keyFile := flag.String("key", "", "Path to TLS server key for optional https")
	flag.Parse()

	m := http.NewServeMux()

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, website) })
	m.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, icon) })

	handler := handlers.LoggingHandler(os.Stdout, m)
	if *certFile != "" && *keyFile != "" {
		log.Printf("Web-Server serves %s webpage on HTTPS port: %s\n", webpage, *tslPort)
		go func() { log.Fatal(ListenAndServeTLSRepSelect(":"+*tslPort, *certFile, *keyFile, handler, rs)) }()
	}
	log.Printf("Web-Server serves %s webpage on HTTP port: %s\n", webpage, *webPort)
	log.Fatalf("%s", ListenAndServeRepSelect(":"+*webPort, handler, rs))
}
