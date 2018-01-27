package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
//	"regexp"
//	"strings"
//	"sync"
//	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	alexa "github.com/waringer/go-alexa/skillserver"
//	mpd "github.com/fhs/gompd/mpd"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var (
	buildstamp string
	githash    string
	Conf       Configuration
	DB         sql.DB
)


type Configuration struct {
        BindingIP    string `json:"bindingIP"`
        BindingPort  uint   `json:"bindingPort"`
        AmazonAppID  string `json:"amazonAppID"`
        PidFile      string `json:"pidFile"`
        StreamURL    string `json:"streamURL"`
        DBUser       string `json:"dbUser"`
        DBPassword   string `json:"dbPassword"`
        DBName       string `json:"dbName"`
//        MpdIP        string `json:"mpdIP"`
//        MpdPort      uint   `json:"mpdPort"`
}

func saveConfig(c Configuration, filename string) error {
    bytes, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }

    return ioutil.WriteFile(filename, bytes, 0644)
}

func loadConfig(filename string) (Configuration, error) {
    DefaultConf := Configuration { BindingPort : 3081, PidFile : "/var/run/alexa_radio.pid" }

    bytes, err := ioutil.ReadFile(filename)
    if err != nil {
        return DefaultConf, err
    }

    err = json.Unmarshal(bytes, &DefaultConf)
    if err != nil {
        return Configuration{}, err
    }

    return DefaultConf, nil
}

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
//		saveConfig(Config, *ConfFile)
	}
	Conf = Config

	if Conf.AmazonAppID == "" {
		log.Fatalln("Amazon AppID fehlt!")
	}

	log.Printf("Alexa-Radio startet %s - %s", buildstamp, githash)

	if Conf.PidFile != "" {
		WritePid(Conf.PidFile)
	}

	//check db
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", Conf.DBUser, Conf.DBPassword, Conf.DBName))
	if err != nil {
	    log.Fatalln("DB Fehler1 : ", err.Error())
	}

	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
	    log.Fatalln("DB Fehler : ", err.Error())
	}
	
	DB = db;

	var Applications = map[string]interface{}{
		"/echo/radio": alexa.EchoApplication{
			AppID:    Conf.AmazonAppID,
			OnIntent: RadioHandler,
			OnLaunch: RadioHandler,
			OnSessionEnded: InfoHandler,
			OnAudioPlayerState: AudioHandler,
			OnException: InfoHandler,
		},
	}

	RunAlexa(Applications, Conf.BindingIP, fmt.Sprintf("%d", Conf.BindingPort))
}

func RunAlexa(apps map[string]interface{}, ip, port string) {
	router := mux.NewRouter()
	alexa.Init(apps, router)

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(ip + ":" + port)
}

func AudioHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
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
		Type : "AudioPlayer.Play",
		PlayBehavior : "REPLACE_ALL",
		AudioItem : &alexa.EchoAudioItem{
		    Stream : alexa.EchoStream {
//			Url : Conf.StreamURL,
			Url : "https://alexa.waringer-atg.de/music/hw/Interpreten/Anne%20Clark/Anne%20Clark%20-%20Sleeper%20in%20Metropolis.mp3",
			Token : "token345",
			OffsetInMilliseconds : 0 } } }

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

func InfoHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
//	fmt.Println(echoReq)
	log.Printf("----> Info Request Typ %s empfangen, UserID %s\n", echoReq.Request.Type, echoReq.Session.User.UserID)
}

func RadioHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
//	fmt.Println(echoReq)
	log.Printf("----> Request für Intent %s empfangen, UserID %s\n", echoReq.Request.Intent.Name, echoReq.Session.User.UserID)
//fmt.Printf("->" + Conf.StreamURL)

	log.Printf("-> intent")

	switch echoReq.Request.Intent.Name {
	case "AMAZON.ResumeIntent":
	    log.Printf("resume intent")
	case "AMAZON.PauseIntent":
	    Directive := alexa.EchoDirective{ Type : "AudioPlayer.Stop" }
	    echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
	    log.Printf("pause intent")
	case "AMAZON.CancelIntent":
	    log.Printf("cancel intent")
	case "AMAZON.StopIntent":
	    Directive := alexa.EchoDirective{ Type : "AudioPlayer.Stop" }
	    echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
	    log.Printf("stop intent")
	case "AMAZON.HelpIntent":
	    log.Printf("help intent")
	default:
	    Directive := alexa.EchoDirective{
		Type : "AudioPlayer.Play",
		PlayBehavior : "REPLACE_ALL",
		AudioItem : &alexa.EchoAudioItem{
		    Stream : alexa.EchoStream {
			Url : Conf.StreamURL,
			Token : "token123",
			OffsetInMilliseconds : 0 } } }

	    echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)

	    log.Printf("default intent")
	}

	log.Printf("intent <-")

	bytes, _ := json.MarshalIndent(echoResp, "", "  ")
	ioutil.WriteFile("response.json", bytes, 0644)

//	log.Printf("<---- Antworte mit %s\n", card)
}

func WritePid(PidFile string) {
	pid := os.Getpid()

	f, err := os.OpenFile(PidFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to create pid file : %v", err)
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("%d", pid))
}
