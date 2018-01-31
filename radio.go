package main

import (
	//"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/codegangsta/negroni"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	alexa "github.com/waringer/go-alexa/skillserver"
)

// Hauptprogramm. Startet den Download des RSS-Feeds und initialisiert den HTTP-Handler.
func main() {
	ConfFile := flag.String("c", "radio.conf", "config file to use")
	Version := flag.Bool("v", false, "prints current version and exit")
	flag.Parse()

	if *Version {
		fmt.Println("Build:", buildstamp)
		fmt.Println("Githash:", githash)
		os.Exit(0)
	}

	Config, err := loadConfig(*ConfFile)
	if err != nil {
		log.Println("can't read conf file", *ConfFile)
		saveConfig(Config, *ConfFile)
	}
	conf = Config

	if conf.AmazonAppID == "" {
		log.Fatalln("Amazon AppID fehlt!")
	}

	log.Printf("Alexa-Radio startet %s - %s", buildstamp, githash)

	if conf.PidFile != "" {
		writePid(conf.PidFile)
	}

	//check db
	db, err := openDB(conf)
	if err != nil {
		log.Fatalln("DB Fehler : ", err.Error())
	}
	defer db.Close()

	database = db

	var Applications = map[string]interface{}{
		"/echo/radio": alexa.EchoApplication{
			AppID:              conf.AmazonAppID,
			OnIntent:           radioHandler,
			OnLaunch:           radioHandler,
			OnSessionEnded:     infoHandler,
			OnAudioPlayerState: audioHandler,
			OnException:        infoHandler,
		},
	}

	runAlexa(Applications, conf.BindingIP, fmt.Sprintf("%d", conf.BindingPort))
}

func runAlexa(apps map[string]interface{}, ip, port string) {
	router := mux.NewRouter()
	alexa.Init(apps, router)

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(ip + ":" + port)
}

func audioHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
	//	fmt.Println(echoReq)
	log.Printf("----> Audio Request Typ %s empfangen, UserID %s\n", echoReq.Request.Type, echoReq.Session.User.UserID)

	switch echoReq.Request.Type {
	case "AudioPlayer.PlaybackStarted":
		log.Printf("type AudioPlayer.PlaybackStarted")
	case "AudioPlayer.PlaybackStopped":
		log.Printf("type AudioPlayer.PlaybackStopped")
	case "AudioPlayer.PlaybackFinished":
		log.Printf("type AudioPlayer.PlaybackFinished")
	case "AudioPlayer.PlaybackNearlyFinished":
		Directive := alexa.EchoDirective{
			Type:         "AudioPlayer.Play",
			PlayBehavior: "REPLACE_ALL",
			AudioItem: &alexa.EchoAudioItem{
				Stream: alexa.EchoStream{
					Url:                  conf.StreamURL,
					Token:                "token345",
					OffsetInMilliseconds: 0}}}

		echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)

		log.Printf("type AudioPlayer.PlaybackNearlyFinished")
		bytesQ, _ := json.MarshalIndent(echoReq, "", "  ")
		ioutil.WriteFile("info_req.json", bytesQ, 0644)

		bytesR, _ := json.MarshalIndent(echoResp, "", "  ")
		ioutil.WriteFile("info_resp.json", bytesR, 0644)
	default:
		bytesQ, _ := json.MarshalIndent(echoReq, "", "  ")
		ioutil.WriteFile("info_unknown_req.json", bytesQ, 0644)

		bytesR, _ := json.MarshalIndent(echoResp, "", "  ")
		ioutil.WriteFile("info_unknown_resp.json", bytesR, 0644)
		log.Printf("type unknown")
	}
}

func infoHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
	//	fmt.Println(echoReq)
	log.Printf("----> Info Request Typ %s empfangen, UserID %s\n", echoReq.Request.Type, echoReq.Session.User.UserID)
}

func radioHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
	//	fmt.Println(echoReq)
	log.Printf("----> Request für Intent %s empfangen, UserID %s\n", echoReq.Request.Intent.Name, echoReq.Session.User.UserID)
	//fmt.Printf("->" + Conf.StreamURL)

	log.Printf("-> intent")

	switch echoReq.Request.Intent.Name {
	case "AMAZON.ResumeIntent":
		log.Printf("resume intent")
	case "AMAZON.PauseIntent":
		Directive := alexa.EchoDirective{Type: "AudioPlayer.Stop"}
		echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
		log.Printf("pause intent")
	case "AMAZON.CancelIntent":
		log.Printf("cancel intent")
	case "AMAZON.StopIntent":
		Directive := alexa.EchoDirective{Type: "AudioPlayer.Stop"}
		echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
		log.Printf("stop intent")
	case "AMAZON.HelpIntent":
		log.Printf("help intent")
	default:
		Directive := alexa.EchoDirective{
			Type:         "AudioPlayer.Play",
			PlayBehavior: "REPLACE_ALL",
			AudioItem: &alexa.EchoAudioItem{
				Stream: alexa.EchoStream{
					Url:                  conf.StreamURL,
					Token:                "token123",
					OffsetInMilliseconds: 0}}}

		echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)

		log.Printf("default intent")
	}

	log.Printf("intent <-")

	bytes, _ := json.MarshalIndent(echoResp, "", "  ")
	ioutil.WriteFile("response.json", bytes, 0644)

	//	log.Printf("<---- Antworte mit %s\n", card)
}
