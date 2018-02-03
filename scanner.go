package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
)

type trackInfo struct {
	fileName   string
	track      string
	trackIndex int
	artist     string
	album      string
	albumIndex int
	found      bool
}

var jobs = make(chan trackInfo, 1000)

func main() {
	ConfFile := flag.String("c", "radio.conf", "config file to use")
	EmptyDB := flag.Bool("e", false, "reset db before insert new")
	Version := flag.Bool("v", false, "prints current version and exit")
	flag.Parse()

	if *Version {
		fmt.Println("Build:", buildstamp)
		fmt.Println("Githash:", githash)
		os.Exit(0)
	}

	Config, err := loadConfig(*ConfFile)
	if err != nil {
		log.Fatalln("can't read conf file", *ConfFile)
	}
	conf = Config

	log.Printf("Alexa-Radio Scanner startet %s - %s", buildstamp, githash)

	//check db
	db, err := openDB(conf)
	if err != nil {
		log.Fatalln("DB Fehler : ", err.Error())
	}
	defer db.Close()

	database = db

	if *EmptyDB {
		emptyDB()
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go dbWorker(&wg)
	}

	for confIndex, actualConf := range conf.Scanner {
		switch actualConf.FileAccessMode {
		case "nfs":
			startNFSScanner(actualConf, confIndex)
		default:
			startScanner(actualConf, confIndex)
		}
	}

	close(jobs)
	log.Println("jobs closed")
	wg.Wait()
	log.Println("All finished")
}

func startNFSScanner(actualConf ScannerConfiguration, confIndex int) {
	log.Println("using nfs for file access")

	nfsMount, err := nfs.DialMount(actualConf.NFSServer)
	defer nfsMount.Close()
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}

	nfsTarget, err := nfsMount.Mount(actualConf.NFSShare, rpc.AuthNull)
	if err != nil {
		defer nfsTarget.Close()
		log.Fatalf("unable to mount volume: %v", err)
	}

	for _, actualPath := range actualConf.IncludePaths {
		scanNFSPath(nfsTarget, confIndex, actualPath, 0)
	}
}

func scanNFSPath(v *nfs.Target, confIndex int, path string, deep int) {
	log.Println("Scanning path :", path)

	dirs, err := v.ReadDirPlus(path)
	if err != nil {
		return
	}

	for _, dir := range dirs {
		fileName := path + "/" + dir.FileName

		if (dir.FileName != ".") && (dir.FileName != "..") && !isExcludedPath(confIndex, fileName) {
			switch {
			case dir.IsDir():
				scanNFSPath(v, confIndex, fileName, deep+1)
			case isValidExtension(confIndex, dir.FileName):
				scanFile(v, confIndex, fileName, "", deep)
			}
		}
	}
}

func startScanner(actualConf ScannerConfiguration, confIndex int) {
	log.Println("using local filesystem for file access")

	for _, actualPath := range actualConf.IncludePaths {
		scanPath(confIndex, actualPath, actualConf.LocalBasePath, 0)
	}
}

func scanPath(confIndex int, path string, basePath string, deep int) {
	log.Println("Scanning path :", basePath, path)

	dirs, err := ioutil.ReadDir(basePath + path)
	if err != nil {
		return
	}

	for _, dir := range dirs {
		fileName := path + "/" + dir.Name()

		if (dir.Name() != ".") && (dir.Name() != "..") && !isExcludedPath(confIndex, fileName) {
			switch {
			case dir.IsDir():
				scanPath(confIndex, fileName, basePath, deep+1)
			case isValidExtension(confIndex, dir.Name()):
				scanFile(nil, confIndex, fileName, basePath, deep)
			}
		}
	}
}

func scanFile(v *nfs.Target, confIndex int, filename string, basePath string, deep int) {
	if conf.Scanner[confIndex].UseTags == true {
		trackinfo := getTags(v, basePath, filename)
		if !trackinfo.found {
			log.Println("Using path matching for deep", deep)
			trackinfo = parseFileName(filename, conf.Scanner[confIndex].Extractors[deep])
		}
		jobs <- trackinfo
	} else {
		log.Println("Using path matching for deep", deep)
		trackinfo := parseFileName(filename, conf.Scanner[confIndex].Extractors[deep])
		jobs <- trackinfo
	}
}

func parseFileName(fileName string, regEx string) trackInfo {
	tags := getParams(regEx, fileName)
	albumid, _ := strconv.Atoi(tags["albumid"])
	trackid, _ := strconv.Atoi(tags["trackid"])

	return trackInfo{
		fileName:   fileName,
		track:      strings.TrimSpace(tags["track"]),
		trackIndex: trackid,
		artist:     strings.TrimSpace(tags["artist"]),
		album:      strings.TrimSpace(tags["album"]),
		albumIndex: albumid,
		found:      true,
	}
}

func getTags(v *nfs.Target, basePath string, fileName string) trackInfo {
	var fileHandle *os.File

	if v != nil {
		tmpfile, err := ioutil.TempFile("", "scanner")
		if err != nil {
			log.Fatal(err)
		}
		defer os.Remove(tmpfile.Name()) // clean up
		defer tmpfile.Close()

		rf, err := v.Open(fileName)
		if err != nil {
			log.Fatalf("Open: %s", err.Error())
		}
		defer rf.Close()

		rfc, _ := ioutil.ReadAll(rf)
		tmpfile.Write(rfc)
		tmpfile.Seek(0, io.SeekStart)

		fileHandle = tmpfile
	} else {
		f, err := os.Open(basePath + fileName)
		if err != nil {
			log.Fatalf("error open file: %s", err.Error())
		}
		defer f.Close()

		fileHandle = f
	}

	m, err := tag.ReadFrom(fileHandle)
	if err != nil {
		log.Println("Can't read tag :", err.Error())
	} else {
		if (len(strings.TrimSpace(m.Artist())) == 0) && (len(strings.TrimSpace(m.Album())) == 0) && (len(strings.TrimSpace(m.Title())) == 0) {
			fmt.Println("-> Tags empty")
		} else {
			trackid, _ := m.Track()

			return trackInfo{
				fileName:   fileName,
				track:      strings.TrimSpace(m.Title()),
				trackIndex: trackid,
				artist:     strings.TrimSpace(m.Artist()),
				album:      strings.TrimSpace(m.Album()),
				albumIndex: m.Year(),
				found:      true,
			}
		}
	}

	return trackInfo{
		fileName:   fileName,
		track:      "",
		trackIndex: 0,
		artist:     "",
		album:      "",
		albumIndex: 0,
		found:      false,
	}
}

func getParams(regEx, fileName string) (paramsMap map[string]string) {
	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(fileName)

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

func isValidExtension(confIndex int, fileName string) bool {
	fileName = strings.TrimSpace(fileName)

	for ext := range conf.Scanner[confIndex].ValidExtensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}

	log.Println("invalid extension of file :", fileName)
	return false
}

func isExcludedPath(confIndex int, path string) bool {
	for _, excludedPath := range conf.Scanner[confIndex].ExcludePaths {
		if strings.HasPrefix(path, excludedPath) {
			log.Println("Excluded path :", path)
			return true
		}
	}

	return false
}

func insertDB(track trackInfo) {
	var artistIndex int
	for { // try multiple time to get id
		err := database.QueryRow("SELECT fnAddArtist(?)", track.artist).Scan(&artistIndex)
		if err != nil {
			log.Println("DB Error ArtisT:", err)
		}
		if artistIndex != -1 {
			break
		}
		time.Sleep(2000)
	}

	var albumIndex int
	for { // try multiple time to get id
		err := database.QueryRow("SELECT fnAddAlbum(?,?)", track.album, track.albumIndex).Scan(&albumIndex)
		if err != nil {
			log.Println("DB Error AlbuM:", err)
		}
		if albumIndex != -1 {
			break
		}
		time.Sleep(2000)
	}

	var trackIndex int
	err := database.QueryRow("SELECT TK_id FROM TracK WHERE TK_FileName = ?", track.fileName).Scan(&trackIndex)
	if err != nil {
		_, err = database.Exec("INSERT INTO TracK (TK_FileName, TK_Name, TK_AT_id, TK_AM_id, TK_Index) VALUES (?,?,?,?,?)", track.fileName, track.track, artistIndex, albumIndex, track.trackIndex)
		if err != nil {
			log.Println("DB Error TracK:", err, track, artistIndex, albumIndex)
		}
	}
}

func emptyDB() {
	log.Println("=> Truncate tables")
	database.Exec("SET FOREIGN_KEY_CHECKS = 0")
	database.Exec("truncate table ArtisT")
	database.Exec("truncate table AlbuM")
	database.Exec("truncate table TracK")
	database.Exec("SET FOREIGN_KEY_CHECKS = 1;")
}

func dbWorker(wg *sync.WaitGroup) {
	for job := range jobs {
		insertDB(job)
	}
	wg.Done()
}
