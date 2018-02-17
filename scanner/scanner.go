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

	"../shared"

	"github.com/dhowden/tag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
)

var (
	buildstamp string
	githash    string

	dbJobs      = make(chan shared.DBJob, 500)
	fileJobs    = make(chan shared.ScanFileInfo, 1000)
	scannerJobs = make(chan shared.ScannerInfo, 50)
	runningJobs = make(chan bool, 10000)
	timeStampDB = ""

	updateDB     = flag.Bool("u", false, "update db entrys if exists")
	simulateOnly = flag.Bool("s", false, "only simulate - no change in the db")
)

func main() {
	confFile := flag.String("c", "radio.conf", "config file to use")
	emptyDB := flag.Bool("e", false, "reset db before insert new")
	version := flag.Bool("v", false, "prints current version and exit")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	flag.Parse()

	if *version {
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

	err := shared.LoadConfig(*confFile)
	if err != nil {
		log.Fatalln("can't read conf file", *confFile)
	}

	log.Printf("> Alexa-Radio Scanner startet %s - %s", buildstamp, githash)

	//check db
	err = shared.OpenDB()
	if err != nil {
		log.Fatalln("DB Fehler : ", err.Error())
	}
	defer shared.CloseDB()

	timeStampDB = shared.GetCurrentDBTimestamp()

	if *emptyDB {
		shared.EmptyDB()
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

	for confIndex, actualConf := range shared.Conf.Scanner {
		scannerJobs <- shared.ScannerInfo{
			ActualConf: actualConf,
			ConfIndex:  confIndex,
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
	for _, actualConf := range shared.Conf.Scanner {
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

func startNFSScanner(scannerConf shared.ScannerInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	log.Println("=> using nfs for file access")

	nfsMount, err := nfs.DialMount(scannerConf.ActualConf.NFSServer)
	defer nfsMount.Close()
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}

	nfsTarget, err := nfsMount.Mount(scannerConf.ActualConf.NFSShare, rpc.AuthNull)
	if err != nil {
		defer nfsTarget.Close()
		log.Fatalf("unable to mount volume: %v", err)
	}

	for _, actualPath := range scannerConf.ActualConf.IncludePaths {
		scanNFSPath(nfsTarget, scannerConf.ConfIndex, actualPath, 0)
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
				fileJobs <- shared.ScanFileInfo{
					V:         v,
					ConfIndex: confIndex,
					Filename:  fileName,
					BasePath:  "",
					Deep:      deep,
				}
			}
		}
	}
}

func startScanner(scannerConf shared.ScannerInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	log.Println("=> using local filesystem for file access")

	for _, actualPath := range scannerConf.ActualConf.IncludePaths {
		scanPath(scannerConf.ConfIndex, actualPath, scannerConf.ActualConf.LocalBasePath, 0)
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
				fileJobs <- shared.ScanFileInfo{
					V:         nil,
					ConfIndex: confIndex,
					Filename:  fileName,
					BasePath:  basePath,
					Deep:      deep,
				}
			}
		}
	}
}

func scanFile(confScanFile shared.ScanFileInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	if shared.ExistsInDB(confScanFile.Filename) {
		if *updateDB {
			dbJobs <- shared.DBJob{
				JobType: "update",
				Track:   getTrackInfo(confScanFile),
			}
		} else {
			dbJobs <- shared.DBJob{
				JobType: "touch",
				Track: shared.TrackInfo{
					FileName: confScanFile.Filename,
					Found:    true,
				},
			}
		}
	} else {
		dbJobs <- shared.DBJob{
			JobType: "insert",
			Track:   getTrackInfo(confScanFile),
		}
	}
}

func getTrackInfo(confScanFile shared.ScanFileInfo) shared.TrackInfo {
	if shared.Conf.Scanner[confScanFile.ConfIndex].UseTags == true {
		trackinfo := getTags(confScanFile.V, confScanFile.BasePath, confScanFile.Filename)
		if !trackinfo.Found {
			log.Println("==> Using path matching for deep:", confScanFile.Deep, confScanFile.Filename)
			trackinfo = parseFileName(confScanFile.Filename, shared.Conf.Scanner[confScanFile.ConfIndex].Extractors[confScanFile.Deep])
		}
		return trackinfo
	}

	log.Println("==> Using path matching for deep:", confScanFile.Deep, confScanFile.Filename)
	return parseFileName(confScanFile.Filename, shared.Conf.Scanner[confScanFile.ConfIndex].Extractors[confScanFile.Deep])
}

func parseFileName(fileName string, regEx string) shared.TrackInfo {
	tags := getParams(regEx, fileName)
	albumid, _ := strconv.Atoi(tags["albumid"])
	trackid, _ := strconv.Atoi(tags["trackid"])

	return shared.TrackInfo{
		FileName:   fileName,
		Track:      strings.TrimSpace(tags["track"]),
		TrackIndex: trackid,
		Artist:     strings.TrimSpace(tags["artist"]),
		Album:      strings.TrimSpace(tags["album"]),
		AlbumIndex: albumid,
		Found:      true,
	}
}

func getTags(v *nfs.Target, basePath string, fileName string) shared.TrackInfo {
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

			return shared.TrackInfo{
				FileName:   fileName,
				Track:      strings.TrimSpace(m.Title()),
				TrackIndex: trackid,
				Artist:     strings.TrimSpace(m.Artist()),
				Album:      strings.TrimSpace(m.Album()),
				AlbumIndex: m.Year(),
				Found:      true,
			}
		}
	}

	return shared.TrackInfo{
		FileName:   fileName,
		Track:      "",
		TrackIndex: 0,
		Artist:     "",
		Album:      "",
		AlbumIndex: 0,
		Found:      false,
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

	for ext := range shared.Conf.Scanner[confIndex].ValidExtensions {
		if strings.HasSuffix(fileName, strings.ToLower(ext)) {
			return true
		}
	}

	log.Println("=> invalid extension of file:", fileName)
	return false
}

func isExcludedPath(confIndex int, path string) bool {
	for _, excludedPath := range shared.Conf.Scanner[confIndex].ExcludePaths {
		if strings.HasPrefix(path, excludedPath) {
			log.Println("=> Excluded path:", path)
			return true
		}
	}

	return false
}

func startRemove(actualConf shared.ScannerConfiguration) {
	if actualConf.RemoveNoLongerExisting {
		for _, actualPath := range actualConf.IncludePaths {
			trackIDs := shared.GetOldTracks(actualPath, actualConf.ExcludePaths, timeStampDB)
			if len(trackIDs) != 0 {
				log.Println("=> found old with ids:", trackIDs)
				for _, trackID := range trackIDs {
					dbJobs <- shared.DBJob{
						JobType: "remove",
						Track: shared.TrackInfo{
							TrackIndex: trackID,
							Found:      true,
						},
					}
				}
			}
		}
	}
}

func dbWorker(wg *sync.WaitGroup) {
	for job := range dbJobs {
		if !*simulateOnly {
			switch job.JobType {
			case "insert":
				runningJobs <- true
				shared.InsertTrack(job.Track)
				_ = <-runningJobs
			case "touch":
				runningJobs <- true
				shared.TouchTrack(job.Track.FileName)
				_ = <-runningJobs
			case "update":
				runningJobs <- true
				shared.UpdateTrack(job.Track)
				_ = <-runningJobs
			case "remove":
				runningJobs <- true
				shared.RemoveTrackDB(job.Track.TrackIndex)
				_ = <-runningJobs
			}
		} else {
			switch job.JobType {
			case "insert":
				log.Println("DB insert of track filename:", job.Track.FileName)
				log.Println("DB insert of track:", job.Track.TrackIndex, job.Track.Track)
				log.Println("DB insert of artist:", job.Track.Artist)
				log.Println("DB insert of album:", job.Track.AlbumIndex, job.Track.Album)
			case "touch":
				log.Println("DB touch of track filename:", job.Track.FileName)
			case "update":
				log.Println("DB update of track filename:", job.Track.FileName)
				log.Println("DB update of track:", job.Track.TrackIndex, job.Track.Track)
				log.Println("DB update of artist:", job.Track.Artist)
				log.Println("DB update of album:", job.Track.AlbumIndex, job.Track.Album)
			case "remove":
				log.Println("DB remove of track:", job.Track.TrackIndex)
			}

		}
	}
	wg.Done()
}

func scannerWorker(wg *sync.WaitGroup) {
	for job := range scannerJobs {
		switch job.ActualConf.FileAccessMode {
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
