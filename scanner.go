package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
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

type scannerInfo struct {
	actualConf ScannerConfiguration
	confIndex  int
}

type scanFileInfo struct {
	v         *nfs.Target
	confIndex int
	filename  string
	basePath  string
	deep      int
}

type dbJob struct {
	jobType string
	track   trackInfo
}

var dbJobs = make(chan dbJob, 500)
var fileJobs = make(chan scanFileInfo, 1000)
var scannerJobs = make(chan scannerInfo, 50)
var runningJobs = make(chan bool, 10000)
var timeStampDB = ""

var updateDB = flag.Bool("u", false, "update db entrys if exists")

func main() {
	ConfFile := flag.String("c", "radio.conf", "config file to use")
	EmptyDB := flag.Bool("e", false, "reset db before insert new")
	Version := flag.Bool("v", false, "prints current version and exit")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	flag.Parse()

	if *Version {
		fmt.Println("Build:", buildstamp)
		fmt.Println("Githash:", githash)
		os.Exit(0)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	Config, err := loadConfig(*ConfFile)
	if err != nil {
		log.Fatalln("can't read conf file", *ConfFile)
	}
	conf = Config

	log.Printf("> Alexa-Radio Scanner startet %s - %s", buildstamp, githash)

	//check db
	db, err := openDB(conf)
	if err != nil {
		log.Fatalln("DB Fehler : ", err.Error())
	}
	defer db.Close()

	database = db
	timeStampDB = getCurrentDBTimestamp()

	if *EmptyDB {
		emptyDB()
	}

	var wgDB sync.WaitGroup
	var wg sync.WaitGroup
	// db insert workers
	for i := 0; i < 20; i++ {
		wgDB.Add(1)
		go dbWorker(&wgDB)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go fileWorker(&wg)
	}

	//scan workers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go scannerWorker(&wg)
	}

	for confIndex, actualConf := range conf.Scanner {
		scannerJobs <- scannerInfo{
			actualConf: actualConf,
			confIndex:  confIndex,
		}
		time.Sleep(100 * time.Millisecond) // give job some time to start
	}

	log.Println("> wait for jobs started")
	time.Sleep(500 * time.Millisecond)

	for {
		if (len(scannerJobs) == 0) && (len(fileJobs) == 0) && (len(dbJobs) == 0) && (len(runningJobs) == 0) {
			break
		}
		time.Sleep(500 * time.Millisecond) // give jobs time to work
	}

	close(scannerJobs)
	close(fileJobs)
	log.Println("> Jobs closed")
	wg.Wait()

	for {
		if (len(dbJobs) == 0) && (len(runningJobs) == 0) {
			break
		}
		time.Sleep(500 * time.Millisecond) // give jobs time to work
	}

	log.Println("> remove old entrys")
	for _, actualConf := range conf.Scanner {
		startRemove(actualConf)
	}

	close(dbJobs)
	wgDB.Wait()

	log.Println("> All finished")

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}

func startNFSScanner(scannerConf scannerInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	log.Println("=> using nfs for file access")

	nfsMount, err := nfs.DialMount(scannerConf.actualConf.NFSServer)
	defer nfsMount.Close()
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}

	nfsTarget, err := nfsMount.Mount(scannerConf.actualConf.NFSShare, rpc.AuthNull)
	if err != nil {
		defer nfsTarget.Close()
		log.Fatalf("unable to mount volume: %v", err)
	}

	for _, actualPath := range scannerConf.actualConf.IncludePaths {
		scanNFSPath(nfsTarget, scannerConf.confIndex, actualPath, 0)
	}
}

func scanNFSPath(v *nfs.Target, confIndex int, path string, deep int) {
	log.Println("=> Scanning path:", path)

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
				//scanFile(v, confIndex, fileName, "", deep)
				fileJobs <- scanFileInfo{
					v:         v,
					confIndex: confIndex,
					filename:  fileName,
					basePath:  "",
					deep:      deep,
				}
			}
		}
	}
}

func startScanner(scannerConf scannerInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	log.Println("=> using local filesystem for file access")

	for _, actualPath := range scannerConf.actualConf.IncludePaths {
		scanPath(scannerConf.confIndex, actualPath, scannerConf.actualConf.LocalBasePath, 0)
	}
}

func scanPath(confIndex int, path string, basePath string, deep int) {
	log.Println("=> Scanning path:", basePath, path)

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
				//scanFile(nil, confIndex, fileName, basePath, deep)
				fileJobs <- scanFileInfo{
					v:         nil,
					confIndex: confIndex,
					filename:  fileName,
					basePath:  basePath,
					deep:      deep,
				}
			}
		}
	}
}

func scanFile(confScanFile scanFileInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	if existsInDB(confScanFile.filename) {
		if *updateDB {
			dbJobs <- dbJob{
				jobType: "update",
				track:   getTrackInfo(confScanFile),
			}
		} else {
			dbJobs <- dbJob{
				jobType: "touch",
				track: trackInfo{
					fileName: confScanFile.filename,
					found:    true,
				},
			}
		}
	} else {
		dbJobs <- dbJob{
			jobType: "insert",
			track:   getTrackInfo(confScanFile),
		}
	}
}

func getTrackInfo(confScanFile scanFileInfo) trackInfo {
	if conf.Scanner[confScanFile.confIndex].UseTags == true {
		trackinfo := getTags(confScanFile.v, confScanFile.basePath, confScanFile.filename)
		if !trackinfo.found {
			log.Println("==> Using path matching for deep:", confScanFile.deep)
			trackinfo = parseFileName(confScanFile.filename, conf.Scanner[confScanFile.confIndex].Extractors[confScanFile.deep])
		}
		return trackinfo
	}

	log.Println("==> Using path matching for deep:", confScanFile.deep)
	return parseFileName(confScanFile.filename, conf.Scanner[confScanFile.confIndex].Extractors[confScanFile.deep])
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
		log.Println("=> Can't read tag:", fileName, err.Error())
	} else {
		if (len(strings.TrimSpace(m.Artist())) == 0) && (len(strings.TrimSpace(m.Album())) == 0) && (len(strings.TrimSpace(m.Title())) == 0) {
			//fmt.Println("-> Tags empty")
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
	fileName = strings.ToLower(strings.TrimSpace(fileName))

	for ext := range conf.Scanner[confIndex].ValidExtensions {
		if strings.HasSuffix(fileName, strings.ToLower(ext)) {
			return true
		}
	}

	log.Println("=> invalid extension of file:", fileName)
	return false
}

func isExcludedPath(confIndex int, path string) bool {
	for _, excludedPath := range conf.Scanner[confIndex].ExcludePaths {
		if strings.HasPrefix(path, excludedPath) {
			log.Println("=> Excluded path:", path)
			return true
		}
	}

	return false
}

func startRemove(actualConf ScannerConfiguration) {
	if actualConf.RemoveNoLongerExisting {
		for _, actualPath := range actualConf.IncludePaths {
			trackIDs := getOldTracks(actualPath, actualConf.ExcludePaths, timeStampDB)
			if len(trackIDs) != 0 {
				log.Println("=> found old with ids:", trackIDs)
				for _, trackID := range trackIDs {
					dbJobs <- dbJob{
						jobType: "remove",
						track: trackInfo{
							trackIndex: trackID,
							found:      true,
						},
					}
				}
			}
		}
	}
}

func dbWorker(wg *sync.WaitGroup) {
	for job := range dbJobs {
		switch job.jobType {
		case "insert":
			insertTrack(job.track)
		case "touch":
			touchTrack(job.track.fileName)
		case "update":
			updateTrack(job.track)
		case "remove":
			removeTrackDB(job.track.trackIndex)
		}
	}
	wg.Done()
}

func scannerWorker(wg *sync.WaitGroup) {
	for job := range scannerJobs {
		switch job.actualConf.FileAccessMode {
		case "nfs":
			startNFSScanner(job)
		default:
			startScanner(job)
		}
	}
	wg.Done()
}

func fileWorker(wg *sync.WaitGroup) {
	for job := range fileJobs {
		scanFile(job)
	}
	wg.Done()
}
