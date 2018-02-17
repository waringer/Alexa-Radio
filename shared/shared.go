package shared

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

var (
	Conf     Configuration
	Database *sql.DB
)

func SaveConfig(filename string) error {
	bytes, err := json.MarshalIndent(Conf, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, bytes, 0644)
}

func LoadConfig(filename string) error {
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
		Conf = DefaultConf
		return err
	}

	err = json.Unmarshal(bytes, &DefaultConf)
	if err != nil {
		Conf = Configuration{}
		return err
	}

	Conf = DefaultConf
	return nil
}

func WritePid() {
	if Conf.PidFile != "" {
		f, err := os.OpenFile(Conf.PidFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatalf("Unable to create pid file : %v", err)
		}
		defer f.Close()

		f.WriteString(fmt.Sprintf("%d", os.Getpid()))
	}
}

func UrlEncode(fileName string) (URL string) {
	URL = Conf.StreamURL
	for _, part := range strings.Split(fileName, "/") {
		URL += "/" + (&url.URL{Path: part}).String()
	}

	return
}
