package shared

import (
	"github.com/vmware/go-nfs-client/nfs"
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

type TrackInfo struct {
	FileName   string
	Track      string
	TrackIndex int
	Artist     string
	Album      string
	AlbumIndex int
	Found      bool
}

type ScannerInfo struct {
	ActualConf ScannerConfiguration
	ConfIndex  int
}

type ScanFileInfo struct {
	V         *nfs.Target
	ConfIndex int
	Filename  string
	BasePath  string
	Deep      int
}

type DBJob struct {
	JobType string
	Track   TrackInfo
}
