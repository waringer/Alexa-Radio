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
	ImportComments         bool                   `json:"importComments"`
	LocalBasePath          string                 `json:"localBasePath"`
	NFSServer              string                 `json:"nfsServer"`
	NFSShare               string                 `json:"nfsShare"`
	ValidExtensions        map[string]interface{} `json:"validExtensions"`
	IncludePaths           []string               `json:"pathIncludes"`
	ExcludePaths           []string               `json:"pathExcludes"`
	Extractors             map[int]string         `json:"tagExtractors"`
	Enabled                bool                   `json:"enabled"`
}

// TrackInfo structure to hold informations about a track
type TrackInfo struct {
	FileName   string
	Track      string
	TrackIndex int
	Artist     string
	Album      string
	AlbumIndex int
	Found      bool
	Comment    string
}

// ScannerInfo structure to hold config values for a scanner task
type ScannerInfo struct {
	ActualConf ScannerConfiguration
	ConfIndex  int
}

// ScanFileInfo structure to hold values for the task of get infos from a file
type ScanFileInfo struct {
	V              *nfs.Target
	ConfIndex      int
	Filename       string
	BasePath       string
	ImportComments bool
	Deep           int
}

// DBJob helper structure for db tasks
type DBJob struct {
	JobType string
	Track   TrackInfo
}

// Responses structure to hold possible responses
type Responses struct {
	NotImplemented []string `json:"notImplemented"`
	Hello          []string `json:"hello"`
	ResumePlay     []string `json:"resumePlay"`
	CantResume     []string `json:"cantResume"`
	PlaylistEnd    []string `json:"playlistEnd"`
	LoopOn         []string `json:"loopOn"`
	LoopOff        []string `json:"loopOff"`
	Searching      []string `json:"searching"`
	CantFind       []string `json:"cantFind"`
}
