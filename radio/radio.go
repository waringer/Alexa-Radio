package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/waringer/Alexa-Radio/shared"

	"github.com/codegangsta/negroni"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	alexa "github.com/waringer/go-alexa/skillserver"
)

var (
	buildstamp string
	githash    string
	responses  shared.Responses
)

func main() {
	confFile := flag.String("c", "radio.conf", "config file to use")
	reponseFile := flag.String("r", "response.conf", "response file to use")
	version := flag.Bool("v", false, "prints current version and exit")
	flag.Parse()

	if *version {
		fmt.Println("Build:", buildstamp)
		fmt.Println("Githash:", githash)
		os.Exit(0)
	}

	err := shared.LoadConfig(*confFile)
	if err != nil {
		log.Println("can't read conf file", *confFile)
		shared.SaveConfig(*confFile)
	}

	err = loadResponses(*reponseFile)
	if err != nil {
		log.Println("can't read response file", *reponseFile)
	}

	if shared.Conf.AmazonAppID == "" {
		log.Fatalln("Amazon AppID fehlt!")
	}

	log.Printf("Alexa-Radio startet %s - %s", buildstamp, githash)

	shared.WritePid()

	//check db
	err = shared.OpenDB()
	if err != nil {
		log.Fatalln("DB Fehler : ", err.Error())
	}
	defer shared.CloseDB()

	var Applications = map[string]interface{}{
		"/echo/radio": alexa.EchoApplication{
			AppID:              shared.Conf.AmazonAppID,
			OnIntent:           radioHandler,
			OnLaunch:           radioHandler,
			OnSessionEnded:     infoHandler,
			OnAudioPlayerState: audioHandler,
			OnException:        infoHandler,
			OnPlaybackController: playbackHandler,
		},
	}

	runAlexa(Applications, shared.Conf.BindingIP, fmt.Sprintf("%d", shared.Conf.BindingPort))
}

func runAlexa(apps map[string]interface{}, ip, port string) {
	router := mux.NewRouter()
	alexa.Init(apps, router)

	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(ip + ":" + port)
}

func playbackHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
        log.Printf("----> PlaybackController Request Typ %s empfangen, UserID %s\n", echoReq.Request, echoReq.Session.User.UserID)
        shared.RegisterDevice(echoReq.Context.System.Device.DeviceId)

        switch echoReq.Request.Type {
        case "PlaybackController.NextCommandIssued":
                if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
                        nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
                        nextFileName := shared.GetTrackFileName(nextTrackID)

                        if nextFileName != "" {
                                directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
                                log.Println("URL:", directive.AudioItem.Stream.Url)
                                echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
                        }
                } else {
                        //inform user that playlist is at end
                        card := fmt.Sprint(getRandomResponse(responses.PlaylistEnd)) // Puh, endlich kann ich ausruhen! Playlist ist durch.
                        speech := fmt.Sprintf("<speak>%s</speak>", card)
                        echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                }
                log.Printf("Next intent")
        case "PlaybackController.PreviousCommandIssued":
                if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
                        playingTrackID := shared.GetPlayingTrackID(echoReq.Context.System.Device.DeviceId)
                        prevTrackID := shared.GetPrevTrackID(echoReq.Context.System.Device.DeviceId, playingTrackID)
                        prevFileName := shared.GetTrackFileName(prevTrackID)
                        log.Println("PLAY:", playingTrackID, "PREV:", prevTrackID, "FILENAME:", prevFileName)

                        if prevFileName != "" {
                                directive := makeAudioPlayDirective(prevFileName, false, prevTrackID)
                                log.Println("URL:", directive.AudioItem.Stream.Url)
                                echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
                        }
                } else {
                        //inform user that playlist is at end
                        card := fmt.Sprint(getRandomResponse(responses.PlaylistEnd)) // Puh, endlich kann ich ausruhen! Playlist ist durch.
                        speech := fmt.Sprintf("<speak>%s</speak>", card)
                        echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                }
                log.Printf("PreviousIntent intent")
        case "PlaybackController.PauseCommandIssued":
                Directive := alexa.EchoDirective{Type: "AudioPlayer.Stop"}
                echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
                log.Printf("Pause intent")
        case "PlaybackController.PlayCommandIssued":
                if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
                        playingTrackID := shared.GetPlayingTrackID(echoReq.Context.System.Device.DeviceId)
                        playingFileName := shared.GetTrackFileName(playingTrackID)

                if playingFileName != "" {
                                directive := makeAudioPlayDirective(playingFileName, false, playingTrackID)
                                log.Println("URL:", directive.AudioItem.Stream.Url)
                                echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
                        }
                } else {
                        //inform user that playlist is at end
                        card := fmt.Sprint(getRandomResponse(responses.PlaylistEnd)) // Puh, endlich kann ich ausruhen! Playlist ist durch.
                        speech := fmt.Sprintf("<speak>%s</speak>", card)
                        echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                }
                log.Printf("StartOver intent")
	}
}

func audioHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
	log.Printf("----> Audio Request Typ %s empfangen, UserID %s\n", echoReq.Request.Type, echoReq.Session.User.UserID)
	shared.RegisterDevice(echoReq.Context.System.Device.DeviceId)

	switch echoReq.Request.Type {
	case "AudioPlayer.PlaybackStarted":
		log.Printf("type AudioPlayer.PlaybackStarted")
		shared.MarkTrackSelected(echoReq.Context.System.Device.DeviceId, extractTrackID(echoReq.Request.Token))
	case "AudioPlayer.PlaybackStopped":
		log.Printf("type AudioPlayer.PlaybackStopped")
		shared.MarkTrackPlayed(echoReq.Context.System.Device.DeviceId, extractTrackID(echoReq.Request.Token))
	case "AudioPlayer.PlaybackFinished":
		log.Printf("type AudioPlayer.PlaybackFinished")
	case "AudioPlayer.PlaybackFailed":
		log.Printf("type AudioPlayer.PlaybackFailed")
		fallthrough
	case "AudioPlayer.PlaybackNearlyFinished":
		if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
			nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
			nextFileName := shared.GetTrackFileName(nextTrackID)
			shared.MarkTrackPlayed(echoReq.Context.System.Device.DeviceId, extractTrackID(echoReq.Request.Token))

			if nextFileName != "" {
				directive := makeAudioPlayDirective(nextFileName, true, nextTrackID)
				log.Println("URL:", directive.AudioItem.Stream.Url)
				echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
			}
		}
		log.Printf("type AudioPlayer.PlaybackNearlyFinished")
	default:
		log.Printf("type unknown")
	}
}

func infoHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
	log.Printf("----> Info Request Typ %s empfangen, UserID %s\n", echoReq.Request.Type, echoReq.Session.User.UserID)
	shared.RegisterDevice(echoReq.Context.System.Device.DeviceId)
}

func radioHandler(echoReq *alexa.EchoRequest, echoResp *alexa.EchoResponse) {
	log.Printf("----> Request für Intent %s empfangen, UserID %s\n", echoReq.Request.Intent.Name, echoReq.Session.User.UserID)
	shared.RegisterDevice(echoReq.Context.System.Device.DeviceId)

	switch echoReq.Request.Intent.Name {
	case "AMAZON.ShuffleOnIntent":
		shared.SwitchShuffle(echoReq.Context.System.Device.DeviceId, true)
		log.Printf("ShuffleOn intent")

		//todo response
		//speech := fmt.Sprint(`<speak>`)
		//card := fmt.Sprint("Shuffle ist jetzt an")
		//speech = fmt.Sprintf("%s%s</speak>", speech, card)
	case "AMAZON.ShuffleOffIntent":
		shared.SwitchShuffle(echoReq.Context.System.Device.DeviceId, false)
		log.Printf("ShuffleOff intent")

		//todo response
		//speech := fmt.Sprint(`<speak>`)
		//card := fmt.Sprint("Shuffle ist jetzt aus")
		//speech = fmt.Sprintf("%s%s</speak>", speech, card)
	case "AMAZON.StartOverIntent":
                if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
                        playingTrackID := shared.GetPlayingTrackID(echoReq.Context.System.Device.DeviceId)
                        playingFileName := shared.GetTrackFileName(playingTrackID)

                if playingFileName != "" {
                                directive := makeAudioPlayDirective(playingFileName, false, playingTrackID)
                                log.Println("URL:", directive.AudioItem.Stream.Url)
                                echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
                        }
                } else {
                        //inform user that playlist is at end
                        card := fmt.Sprint(getRandomResponse(responses.PlaylistEnd)) // Puh, endlich kann ich ausruhen! Playlist ist durch.
                        speech := fmt.Sprintf("<speak>%s</speak>", card)
                        echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                }
		log.Printf("StartOver intent")

	case "AMAZON.RepeatIntent":
		log.Printf("Repeat intent")

		//todo implement
		card := fmt.Sprint(getRandomResponse(responses.NotImplemented)) // Stör mich jetzt nicht, ich hab zu tun!
		speech := fmt.Sprintf("<speak>%s</speak>", card)
		echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
	case "AMAZON.LoopOnIntent":
		shared.SwitchLoop(echoReq.Context.System.Device.DeviceId, true)
		log.Printf("LoopOff intent")

		card := fmt.Sprint(getRandomResponse(responses.LoopOn)) // Loop ist jetzt an
		speech := fmt.Sprintf("<speak>%s</speak>", card)
		echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
	case "AMAZON.LoopOffIntent":
		shared.SwitchLoop(echoReq.Context.System.Device.DeviceId, false)
		log.Printf("LoopOff intent")

		card := fmt.Sprint(getRandomResponse(responses.LoopOff)) // Loop ist jetzt aus
		speech := fmt.Sprintf("<speak>%s</speak>", card)
		echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
	case "AMAZON.PauseIntent":
		Directive := alexa.EchoDirective{Type: "AudioPlayer.Stop"}
		echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
		log.Printf("Pause intent")
	case "AMAZON.CancelIntent":
		log.Printf("Cancel intent")
	case "AMAZON.StopIntent":
		Directive := alexa.EchoDirective{Type: "AudioPlayer.Stop"}
		echoResp.Response.Directives = append(echoResp.Response.Directives, Directive)
		log.Printf("Stop intent")
	case "AMAZON.NextIntent":
		log.Println("abcd: NEXT")
		if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
			nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
			nextFileName := shared.GetTrackFileName(nextTrackID)

			if nextFileName != "" {
				directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
				log.Println("URL:", directive.AudioItem.Stream.Url)
				echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
			}
		} else {
			//inform user that playlist is at end
			card := fmt.Sprint(getRandomResponse(responses.PlaylistEnd)) // Puh, endlich kann ich ausruhen! Playlist ist durch.
			speech := fmt.Sprintf("<speak>%s</speak>", card)
			echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
		}
		log.Printf("Next intent")
	case "AMAZON.PreviousIntent":
		if !shared.ShouldStopPlaying(echoReq.Context.System.Device.DeviceId) {
			playingTrackID := shared.GetPlayingTrackID(echoReq.Context.System.Device.DeviceId)
			prevTrackID := shared.GetPrevTrackID(echoReq.Context.System.Device.DeviceId, playingTrackID)
			prevFileName := shared.GetTrackFileName(prevTrackID)
			log.Println("PLAY:", playingTrackID, "PREV:", prevTrackID, "FILENAME:", prevFileName)

			if prevFileName != "" {
				directive := makeAudioPlayDirective(prevFileName, false, prevTrackID)
				log.Println("URL:", directive.AudioItem.Stream.Url)
				echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
			}
		} else {
			//inform user that playlist is at end
			card := fmt.Sprint(getRandomResponse(responses.PlaylistEnd)) // Puh, endlich kann ich ausruhen! Playlist ist durch.
			speech := fmt.Sprintf("<speak>%s</speak>", card)
			echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
		}
		log.Printf("PreviousIntent intent")
	case "AMAZON.HelpIntent":
		log.Printf("Help intent")
		fallthrough


        case "StartPlayList":
                log.Printf("StartPlayList intent")
                SearchString := strings.TrimSpace(echoReq.Request.Intent.Slots["PlayListName"].Value)

                if SearchString != "" {
                        shared.UpdateActualPlayList(echoReq.Context.System.Device.DeviceId, SearchString)
                        nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
                        nextFileName := shared.GetTrackFileName(nextTrackID)

                        shared.SwitchShuffle(echoReq.Context.System.Device.DeviceId, true)

                        if nextFileName != "" {
                                directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
                                log.Println("URL:", directive.AudioItem.Stream.Url)

                                card := fmt.Sprintf(getRandomResponse(responses.Searching), SearchString) // ich such ja schon %s raus
                                speech := fmt.Sprintf("<speak>%s</speak>", card)
                                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

                                echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
                         } else {
                                card := fmt.Sprintf(getRandomResponse(responses.CantFind), SearchString) // Ich konnte für %s absolut nix finden!
                                speech := fmt.Sprintf("<speak>%s</speak>", card)
                                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        }
                    }

        case "ListPlayList":
		log.Println("====> all available PlayLists")

		playLists := shared.GetPlayListNames(echoReq.Context.System.Device.DeviceId)
		log.Printf("PlayListNames: ", playLists)
		card := "Es gibt folgende Playlisten:"
		for _, playList := range playLists {
			card += fmt.Sprintf(" %s ", playList)
			log.Printf("PlayListName: ", playList)
			}
		speech := fmt.Sprintf("<speak>%s</speak>", card)
                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)



	case "AddPlayList":
                log.Println("====> AddPlayList")
		/* SearchString := strings.TrimSpace(echoReq.Request.Intent.Slots["PlayListName"].Value)

                 if SearchString != "" {
                        shared.AddPlayList(SearchString)
                    } */

	case "RemovePlayList":
                log.Println("====> RemovePlayList")
		/* SearchString := "" //strings.TrimSpace(echoReq.Request.Intent.Slots["PlayListName"].Value)

                 if SearchString != "" {
                        shared.DelPlayList(SearchString)
                    } */

        case "AddToPlayList":
                log.Println("====> AddToPlayList")
                SearchString := strings.TrimSpace(echoReq.Request.Intent.Slots["PlayListName"].Value)
		// get PlaylisT ID and Name by sounds like query
		PT_id, PT_Name := shared.GetPlayListInfo(SearchString)
		log.Printf("PlayListName: ", SearchString, PT_Name, PT_id)

                if PT_Name != "" {
			// get TrackID and Filename from actual title
                        TrackID := shared.GetTrackID(echoReq.Context.System.Device.DeviceId)
                        FileName := shared.GetTrackFileName(TrackID)

			log.Printf("TrackID: ", TrackID)
			log.Printf("FileName: ", FileName)
                        if FileName != "" {
				shared.AddToPlayList(PT_id, FileName)
				// add PlaylistItem
                                card := fmt.Sprintf("erledigt") // ich such ja schon %s raus
                                speech := fmt.Sprintf("<speak>%s</speak>", card)
                                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        } else {
                                card := fmt.Sprintf("kein Titel ausgewählt")
                                speech := fmt.Sprintf("<speak>%s</speak>", card)
                                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        }
                  } else {
			  card := fmt.Sprintf("Playlist: ", SearchString, " nicht gefunden") // Ich konnte für %s absolut nix finden!
                           speech := fmt.Sprintf("<speak>%s</speak>", card)
                           echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        }

	case "RemoveFromPlayList":
                log.Println("====> RemoveFromPlayList")
                SearchString := strings.TrimSpace(echoReq.Request.Intent.Slots["PlayListName"].Value)
                // get PlaylisT ID and Name by sounds like query
                PT_id, PT_Name := shared.GetPlayListInfo(SearchString)
                log.Printf("PlayListName: ", SearchString, PT_Name, PT_id)

                if PT_Name != "" {
                        // get TrackID and Filename from actual title
                        TrackID := shared.GetTrackID(echoReq.Context.System.Device.DeviceId)
                        FileName := shared.GetTrackFileName(TrackID)

                        log.Printf("TrackID: ", TrackID)
                        log.Printf("FileName: ", FileName)
                        if FileName != "" {
                                shared.RemoveFromPlayList(PT_id, FileName)
                                // add PlaylistItem
                                card := fmt.Sprintf("erledigt") // ich such ja schon %s raus
                                speech := fmt.Sprintf("<speak>%s</speak>", card)
                                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        } else {
                                card := fmt.Sprintf("kein Titel ausgewählt")
                                speech := fmt.Sprintf("<speak>%s</speak>", card)
                                echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        }
                  } else {
                          card := fmt.Sprintf("Playlist: ", SearchString, " nicht gefunden") // Ich konnte für %s absolut nix finden!
                           speech := fmt.Sprintf("<speak>%s</speak>", card)
                           echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                        }


	case "CurrentlyPlaying":
		log.Printf("CurrentlyPlaying intent")

		Artist, Album, Track := shared.GetPlayingInfo(echoReq.Context.System.Device.DeviceId)
		card := ""

		if (Artist == "") && (Album == "") && (Track == "") {
			card = fmt.Sprint("Ich hab keinen plan was da laufen sollte, oder wieso!")
		} else {
			card = "Das sollte "

			if Track != "" {
				card += fmt.Sprintf(" %s ", Track)
			}
			if Artist != "" {
				card += fmt.Sprintf("von %s ", Artist)
			}
			if Album != "" {
				card += fmt.Sprintf("aus dem album %s ", Album)
			}

			card += " sein "
		}
		speech := fmt.Sprintf("<speak>%s</speak>", card)
		echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
	case "AMAZON.ResumeIntent":
		log.Printf("Resume intent")
		fallthrough
	case "ResumePlay":
		log.Printf("ResumePlay intent")
		nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
		nextFileName := shared.GetTrackFileName(nextTrackID)

		if nextFileName != "" {
			directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
			log.Println("URL:", directive.AudioItem.Stream.Url)

			card := fmt.Sprint(getRandomResponse(responses.ResumePlay)) // Ok, ich mach ja schon weiter!
			speech := fmt.Sprintf("<speak>%s</speak>", card)
			echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

			echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
		} else {
			card := fmt.Sprintf(getRandomResponse(responses.CantResume)) // Hey, erst musst du mir sagen was ich raussuchen muss! Also was soll es sein?
			speech := fmt.Sprintf("<speak>%s</speak>", card)
			echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

			echoResp.Response.ShouldEndSession = false
		}
	case "StartPlay":
		log.Printf("StartPlay intent")
		SearchString := strings.TrimSpace(echoReq.Request.Intent.Slots["Searching"].Value)
		log.Println("====>", SearchString)

		if SearchString != "" {
			shared.UpdateActualPlaying(echoReq.Context.System.Device.DeviceId, SearchString)
			nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
			nextFileName := shared.GetTrackFileName(nextTrackID)

			if nextFileName != "" {
				directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
				log.Println("URL:", directive.AudioItem.Stream.Url)

				card := fmt.Sprintf(getRandomResponse(responses.Searching), SearchString) // ich such ja schon %s raus
				speech := fmt.Sprintf("<speak>%s</speak>", card)
				echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

				echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
			} else {
				card := fmt.Sprintf(getRandomResponse(responses.CantFind), SearchString) // Ich konnte für %s absolut nix finden!
				speech := fmt.Sprintf("<speak>%s</speak>", card)
				echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
			}
		}
	case "StartPlayMusic":
                log.Printf("StartPlayMusic intent")
                log.Println("====> all files in music-folder")

                shared.UpdateActualPlayingMusic(echoReq.Context.System.Device.DeviceId, "")
                nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
                nextFileName := shared.GetTrackFileName(nextTrackID)

                shared.SwitchShuffle(echoReq.Context.System.Device.DeviceId, true)

                if nextFileName != "" {
                        directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
                        log.Println("URL:", directive.AudioItem.Stream.Url)

                        card := fmt.Sprintf(getRandomResponse(responses.Searching), "Music") // ich such ja schon %s raus
                        speech := fmt.Sprintf("<speak>%s</speak>", card)
                        echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

                        echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
                } else {
                        card := fmt.Sprintf(getRandomResponse(responses.CantFind), "Music") // Ich konnte für %s absolut nix finden!
                        speech := fmt.Sprintf("<speak>%s</speak>", card)
                        echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
                }


	case "StartPlayAlbumOrTitle":
		log.Printf("StartPlayAlbumOrTitle intent")
		SEARCHAlbumOrTitle := strings.TrimSpace(echoReq.Request.Intent.Slots["AlbumOrTitle"].Value)
		SEARCHArtistName := strings.TrimSpace(echoReq.Request.Intent.Slots["ArtistName"].Value)
		SearchString := SEARCHAlbumOrTitle + " von " + SEARCHArtistName
		log.Println("====>", SearchString)
		log.Println("====>", SEARCHAlbumOrTitle, SEARCHArtistName)

		if SearchString != "" {
			shared.UpdateActualPlayingAlbumOrTitle(echoReq.Context.System.Device.DeviceId, SEARCHAlbumOrTitle, SEARCHArtistName)
			nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
			nextFileName := shared.GetTrackFileName(nextTrackID)

			if nextFileName != "" {
				directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
				log.Println("URL:", directive.AudioItem.Stream.Url)

				card := fmt.Sprintf(getRandomResponse(responses.Searching), SearchString) // ich such ja schon %s raus
				speech := fmt.Sprintf("<speak>%s</speak>", card)
				echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

				echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
			} else {
				card := fmt.Sprintf(getRandomResponse(responses.CantFind), SearchString) // Ich konnte für %s absolut nix finden!
				speech := fmt.Sprintf("<speak>%s</speak>", card)
				echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
			}
		}
	case "StartPlayEpisode":
		log.Printf("StartPlayEpisode intent")
		SEARCHEpisode := strings.TrimSpace(echoReq.Request.Intent.Slots["Episode"].Value)
		SEARCHArtistName := strings.TrimSpace(echoReq.Request.Intent.Slots["ArtistName"].Value)
		SearchString := "\"" + SEARCHEpisode + "\"" + " von " + SEARCHArtistName
		log.Println("====>", SearchString)
		log.Println("====>", SEARCHEpisode, SEARCHArtistName)

		shared.SwitchShuffle(echoReq.Context.System.Device.DeviceId, false)

		if SearchString != "" {
			shared.UpdateActualPlayingEpisode(echoReq.Context.System.Device.DeviceId, SEARCHEpisode, SEARCHArtistName)
			nextTrackID := shared.GetNextTrackID(echoReq.Context.System.Device.DeviceId)
			nextFileName := shared.GetTrackFileName(nextTrackID)

			if nextFileName != "" {
				directive := makeAudioPlayDirective(nextFileName, false, nextTrackID)
				log.Println("URL:", directive.AudioItem.Stream.Url)

				card := fmt.Sprintf(getRandomResponse(responses.Searching), SearchString) // ich such ja schon %s raus
				speech := fmt.Sprintf("<speak>%s</speak>", card)
				echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

				echoResp.Response.Directives = append(echoResp.Response.Directives, directive)
			} else {
				card := fmt.Sprintf(getRandomResponse(responses.CantFind), SearchString) // Ich konnte für %s absolut nix finden!
				speech := fmt.Sprintf("<speak>%s</speak>", card)
				echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)
			}
		} else {
			goto AskUser
		}
		break
	AskUser:
		fallthrough
	default:
		fallthrough
	case "":
		log.Printf("default intent")

		card := fmt.Sprint(getRandomResponse(responses.Hello)) // Was willst du schon wieder?
		speech := fmt.Sprintf("<speak>%s</speak>", card)
		echoResp.OutputSpeechSSML(speech).Card("Network Music Player", card)

		echoResp.Response.ShouldEndSession = false
	}
}

func makeAudioPlayDirective(fileName string, enqueu bool, trackID int) alexa.EchoDirective {
	playBehavior := "REPLACE_ALL"

	if enqueu {
		playBehavior = "REPLACE_ENQUEUED"
	}
        Artist, Album, Track := shared.GetPlayingInfoTrackID(trackID)
	Album = Album + ""
	return alexa.EchoDirective{
		Type:         "AudioPlayer.Play",
		PlayBehavior: playBehavior,
		AudioItem: &alexa.EchoAudioItem{
			Stream: alexa.EchoStream{
				Url:                  shared.UrlEncode(fileName),
				Token:                fmt.Sprintf("NMP~%d~%s", trackID, time.Now().Format("20060102T150405999999")),
                                OffsetInMilliseconds: 0},
                        Metadata: alexa.AudioItemMetadata{
                                                Title: Track,
                                                Subtitle: Artist}}}
}

func extractTrackID(Token string) (TKid int) {
	var regEx = regexp.MustCompile(`(?:NMP~)(?P<TKid>\d+)~.*`)

	if len(regEx.FindStringIndex(Token)) > 0 {
		TKid, _ = strconv.Atoi(regEx.FindStringSubmatch(Token)[1])
	}

	return
}

func loadResponses(filename string) error {
	DefaultResponses := shared.Responses{
		NotImplemented: []string{"Stör mich jetzt nicht, ich hab zu tun!"},
		Hello:          []string{"Was willst du schon wieder?"},
		ResumePlay:     []string{"Ok, ich mach ja schon weiter!"},
		CantResume:     []string{"Hey, erst musst du mir sagen was ich raussuchen muss! Also was soll es sein?"},
		PlaylistEnd:    []string{"Puh, endlich kann ich ausruhen! Playlist ist durch."},
		LoopOn:         []string{"Loop ist jetzt an"},
		LoopOff:        []string{"Loop ist jetzt aus"},
		Searching:      []string{"ich such ja schon %s raus"},
		CantFind:       []string{"Ich konnte für %s absolut nix finden!"},
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		responses = DefaultResponses
		return err
	}

	err = json.Unmarshal(bytes, &DefaultResponses)
	if err != nil {
		responses = shared.Responses{}
		return err
	}

	responses = DefaultResponses
	return nil
}

func getRandomResponse(responses []string) string {
	rand.Seed(time.Now().Unix())
	return responses[rand.Intn(len(responses))]
}
