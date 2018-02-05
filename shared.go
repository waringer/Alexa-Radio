package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Configuration structure to store the configuration
type Configuration struct {
	BindingIP   string                 `json:"bindingIP"`
	BindingPort uint                   `json:"bindingPort"`
	AmazonAppID string                 `json:"amazonAppID"`
	PidFile     string                 `json:"pidFile"`
	StreamURL   string                 `json:"streamURL"`
	DBUser      string                 `json:"dbUser"`
	DBPassword  string                 `json:"dbPassword"`
	DBName      string                 `json:"dbName"`
	DBServer    string                 `json:"dbServer"`
	Scanner     []ScannerConfiguration `json:"scannerConfiguration"`
}

// ScannerConfiguration structure to store the configuration that apply only to the scanner
type ScannerConfiguration struct {
	UseTags                bool                   `json:"useTags"`
	FileAccessMode         string                 `json:"fileAccessMode"` // nfs or local
	RemoveNoLongerExisting bool                   `json:"removeNoLongerExisting"`
	LocalBasePath          string                 `json:"localBasePath"`
	NFSServer              string                 `json:"nfsServer"`
	NFSShare               string                 `json:"nfsShare"`
	ValidExtensions        map[string]interface{} `json:"validExtensions"`
	IncludePaths           []string               `json:"pathIncludes"`
	ExcludePaths           []string               `json:"pathExcludes"`
	Extractors             map[int]string         `json:"tagExtractors"`
}

var (
	buildstamp string
	githash    string
	conf       Configuration
	database   *sql.DB
)

func saveConfig(c Configuration, filename string) error {
	bytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, bytes, 0644)
}

func loadConfig(filename string) (Configuration, error) {
	DefaultConf := Configuration{
		BindingPort: 3081,
		PidFile:     "/var/run/alexa_radio.pid",
		DBServer:    "localhost",
		Scanner: []ScannerConfiguration{
			ScannerConfiguration{
				UseTags:                true,
				FileAccessMode:         "local",
				RemoveNoLongerExisting: false,
				ValidExtensions: map[string]interface{}{
					".mp3":  true,
					".flac": true,
					".ogg":  true,
				},
				IncludePaths: []string{"/"},
				Extractors: map[int]string{
					2: `.*\/(?P<artist>.*)\/(?P<album>.*)\/(?P<filename>.*)`,
					3: `.*\/(?P<artist>.*)\/(?P<album>.*)\/.*\/(?P<filename>.*)`,
				},
			},
		},
	}

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

func writePid(PidFile string) {
	pid := os.Getpid()

	f, err := os.OpenFile(PidFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to create pid file : %v", err)
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("%d", pid))
}

func urlEncode(fileName string) (URL string) {
	URL = conf.StreamURL
	for _, part := range strings.Split(fileName, "/") {
		URL += "/" + (&url.URL{Path: part}).String()
	}

	return
}
