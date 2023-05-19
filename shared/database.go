package shared

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func OpenDB() error {
	Database = nil
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", Conf.DBUser, Conf.DBPassword, Conf.DBServer, Conf.DBName))
	if err != nil {
		return err
	}

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		return err
	}

	Database = db
	return nil
}

func CloseDB() {
	Database.Close()
}

func EmptyDB() {
	log.Println("=> reset database")
	Database.Exec("SET FOREIGN_KEY_CHECKS = 0")
	Database.Exec("truncate table ActualPlaying")
	Database.Exec("UPDATE DeVice SET DV_LastTKid = NULL")
	Database.Exec("truncate table ArtisT")
	Database.Exec("truncate table AlbuM")
	Database.Exec("truncate table TracK")
	Database.Exec("SET FOREIGN_KEY_CHECKS = 1;")
}

func InsertTrack(track TrackInfo) {
	artistIndex := getArtistID(track.Artist)
	albumIndex := getAlbumID(track.Album, track.AlbumIndex)

	var trackIndex int
	err := Database.QueryRow("SELECT TK_id FROM TracK WHERE TK_FileName = ?", track.FileName).Scan(&trackIndex)
	if err != nil {
		_, err = Database.Exec("INSERT INTO TracK (TK_FileName, TK_Name, TK_AT_id, TK_AM_id, TK_Index, TK_Comment) VALUES (?,?,?,?,?,?)", track.FileName, track.Track, artistIndex, albumIndex, track.TrackIndex, track.Comment)
		if err != nil {
			log.Println("DB Error InsertTrack:", err, track, artistIndex, albumIndex)
		}

		log.Println("==> New Track inserted:", track.Artist, track.Album, track.Track)
	}
}

func UpdateTrack(track TrackInfo) {
	artistIndex := getArtistID(track.Artist)
	albumIndex := getAlbumID(track.Album, track.AlbumIndex)

	_, err := Database.Exec("UPDATE TracK SET TK_Name = ?, TK_AT_id = ?, TK_AM_id = ?, TK_Index = ?, TK_Comment = ?, TK_LastSeen = CURRENT_TIMESTAMP WHERE TK_FileName = ?", track.Track, artistIndex, albumIndex, track.TrackIndex, track.Comment, track.FileName)
	if err != nil {
		log.Println("DB Error UpdateTrack:", err, track, artistIndex, albumIndex)
	}

	log.Println("==> New Track inserted:", track.Artist, track.Album, track.Track)
}

func getArtistID(artist string) (artistIndex int) {
	for { // try multiple time to get id
		err := Database.QueryRow("SELECT fnAddArtist(?)", strings.TrimSpace(artist)).Scan(&artistIndex)
		if err != nil {
			log.Println("DB Error getArtistID:", err)
		}
		if artistIndex != -1 {
			return
		}
		time.Sleep(2000 * time.Millisecond)
		log.Println("retry artist")
	}
}

func getAlbumID(album string, index int) (albumIndex int) {
	for { // try multiple time to get id
		err := Database.QueryRow("SELECT fnAddAlbum(?,?)", strings.TrimSpace(album), index).Scan(&albumIndex)
		if err != nil {
			log.Println("DB Error getAlbumID:", err)
		}
		if albumIndex != -1 {
			return
		}
		time.Sleep(2000 * time.Millisecond)
		log.Println("retry album")
	}
}

func ExistsInDB(fileName string) bool {
	var trackIndex int
	err := Database.QueryRow("SELECT TK_id FROM TracK WHERE TK_FileName = ?", fileName).Scan(&trackIndex)
	return err == nil
}

func RemoveTrackDB(id int) {
	_, err := Database.Exec("CALL spDeleteTrack(?)", id)
	if err != nil {
		log.Println("DB Error RemoveTrackDB:", err, id)
	}
}

func TouchTrack(fileName string) {
	_, err := Database.Exec("UPDATE TracK SET TK_LastSeen = CURRENT_TIMESTAMP WHERE TK_FileName = ?", fileName)
	if err != nil {
		log.Println("DB Error TouchTrack:", err)
	}
}

func GetCurrentDBTimestamp() (stamp string) {
	err := Database.QueryRow("SELECT CURRENT_TIMESTAMP").Scan(&stamp)
	if err != nil {
		log.Println("DB Error GetCurrentDBTimestamp:", err)
	}
	return
}

func GetOldTracks(includePath string, exludePaths []string, stamp string) []int {
	sql := "SELECT TK_id FROM TracK WHERE (TK_LastSeen < ?) AND (TK_FileName LIKE ?) "
	var params []interface{}

	params = append(params, stamp)
	params = append(params, includePath+"%")

	for _, exludePath := range exludePaths {
		sql += " AND (TK_FileName NOT LIKE ?) "
		params = append(params, exludePath+"%")
	}

	rows, err := Database.Query(sql, params...)
	if err != nil {
		log.Println("DB Error GetOldTracks:", err, sql, params)
	} else {
		defer rows.Close()
		var back []int
		for rows.Next() {
			var ret int
			err := rows.Scan(&ret)
			if err != nil {
				log.Println("DB Error GetOldTracks:", err)
			} else {
				back = append(back, ret)
			}
		}
		return back
	}

	return nil
}

func RegisterDevice(deviceID string) {
	_, err := Database.Exec("INSERT INTO DeVice (DV_id, DV_Alias, DV_LastActive) VALUES (?,null, CURRENT_TIMESTAMP) ON DUPLICATE KEY UPDATE DV_LastActive=CURRENT_TIMESTAMP;", deviceID)
	if err != nil {
		log.Println("DB Error RegisterDevice:", err)
	}
}

func UpdateActualPlaying(deviceID, searching string) {
	_, err := Database.Exec("CALL spUpdateActualPlaying(?,?);", deviceID, searching)
	if err != nil {
		log.Println("DB Error UpdateActualPlaying:", err)
	}
}

func UpdateActualPlayingMusic(deviceID, searching string) {
        _, err := Database.Exec("CALL spUpdateActualPlayingMusic(?);", deviceID)
        if err != nil {
                log.Println("DB Error UpdateActualPlayingMusic:", err)
        }
}

func UpdateActualPlayingAlbumOrTitle(deviceID, searchAlbumOrTitle string, searchArtist string) {
        _, err := Database.Exec("CALL spUpdateActualPlayingAlbumOrTitle(?,?,?);", deviceID, searchAlbumOrTitle, searchArtist)
        if err != nil {
                log.Println("DB Error UpdateActualPlayingAlbumOrTitle:", err)
        }
}

func GetNextTrackID(deviceID string) (TrackID int) {
	isRandom := 0

	if getShuffleStatus(deviceID) {
		isRandom = 1
	}

	err := Database.QueryRow("SELECT fnGetNextTrackId(?, ?);", deviceID, isRandom).Scan(&TrackID)
	if err != nil {
		log.Println("DB Error GetNextTrackID:", err)
	}

	return
}

func GetTrackFileName(TrackID int) (FileName string) {
	if TrackID < 0 {
		return
	}

	err := Database.QueryRow("SELECT TK_FileName FROM TracK WHERE TK_id = ?;", TrackID).Scan(&FileName)
	if err != nil {
		log.Println("DB Error GetTrackFileName:", err, TrackID)
	}

	return
}

func MarkTrackPlayed(deviceID string, TrackID int) {
	_, err := Database.Exec("CALL spMarkTackPlayed(?, ?)", deviceID, TrackID)
	if err != nil {
		log.Println("DB Error MarkTrackPlayed:", err, deviceID, TrackID)
	}
}

func GetPlayingInfo(deviceID string) (Artist, Album, Trackname string) {
	err := Database.QueryRow("SELECT AT_Name, AM_Name, TK_Name FROM vTrackInfo INNER JOIN DeVice ON DV_LastTKid = TK_id WHERE DV_id = ?;", deviceID).Scan(&Artist, &Album, &Trackname)
	if err != nil {
		log.Println("DB Error GetPlayingInfo:", err)
	}

	Artist = strings.TrimSpace(Artist)
	Album = strings.TrimSpace(Album)
	Trackname = strings.TrimSpace(Trackname)

	return
}

func SwitchShuffle(deviceID string, shuffle bool) {
	shuffleBit := 0

	if shuffle {
		shuffleBit = 1
	}

	_, err := Database.Exec("UPDATE DeVice SET DV_Shuffle = ? WHERE DV_id = ?;", shuffleBit, deviceID)
	if err != nil {
		log.Println("DB Error SwitchShuffle:", err)
	}
}

func getShuffleStatus(deviceID string) bool {
	var shuffleBit int
	err := Database.QueryRow("SELECT DV_Shuffle FROM DeVice WHERE DV_id = ?", deviceID).Scan(&shuffleBit)
	if err != nil {
		log.Println("DB Error getShuffleStatus:", err)
		return false
	}
	return shuffleBit == 1
}

func SwitchLoop(deviceID string, loop bool) {
	loopBit := 0

	if loop {
		loopBit = 1
	}

	_, err := Database.Exec("UPDATE DeVice SET DV_Loop = ? WHERE DV_id = ?;", loopBit, deviceID)
	if err != nil {
		log.Println("DB Error SwitchLoop:", err)
	}
}

func ShouldStopPlaying(deviceID string) bool {
	if getLoopStatus(deviceID) {
		return false
	}

	rows, err := Database.Query("SELECT DISTINCT AP_Playcount FROM ActualPlaying WHERE AP_DV_id = ? ORDER BY AP_Playcount", deviceID)
	if err != nil {
		log.Println("DB Error ShouldStopPlaying:", err)
		return true
	}

	defer rows.Close()
	var rowcount int
	for rows.Next() {
		rowcount++
	}
	return rowcount <= 1
}

func getLoopStatus(deviceID string) bool {
	var loopBit int
	err := Database.QueryRow("SELECT DV_Loop FROM DeVice WHERE DV_id = ?", deviceID).Scan(&loopBit)
	if err != nil {
		log.Println("DB Error getLoopStatus:", err)
		return false
	}
	return loopBit == 1
}
