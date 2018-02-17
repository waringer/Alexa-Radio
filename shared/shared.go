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

func SaveConfig(c Configuration, filename string) error {
	bytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, bytes, 0644)
}

func LoadConfig(filename string) (Configuration, error) {
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

func WritePid(PidFile string) {
	pid := os.Getpid()

	f, err := os.OpenFile(PidFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to create pid file : %v", err)
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("%d", pid))
}

func UrlEncode(fileName string) (URL string) {
	URL = Conf.StreamURL
	for _, part := range strings.Split(fileName, "/") {
		URL += "/" + (&url.URL{Path: part}).String()
	}

	return
}
