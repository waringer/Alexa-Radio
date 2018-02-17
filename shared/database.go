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
		_, err = Database.Exec("INSERT INTO TracK (TK_FileName, TK_Name, TK_AT_id, TK_AM_id, TK_Index) VALUES (?,?,?,?,?)", track.FileName, track.Track, artistIndex, albumIndex, track.TrackIndex)
		if err != nil {
			log.Println("DB Error TracK:", err, track, artistIndex, albumIndex)
		}

		log.Println("==> New Track inserted:", track.Artist, track.Album, track.Track)
	}
}

func UpdateTrack(track TrackInfo) {
	artistIndex := getArtistID(track.Artist)
	albumIndex := getAlbumID(track.Album, track.AlbumIndex)

	_, err := Database.Exec("UPDATE TracK SET TK_Name = ?, TK_AT_id = ?, TK_AM_id = ?, TK_Index = ?, TK_LastSeen = CURRENT_TIMESTAMP WHERE TK_FileName = ?", track.Track, artistIndex, albumIndex, track.TrackIndex, track.FileName)
	if err != nil {
		log.Println("DB Error TracK:", err, track, artistIndex, albumIndex)
	}

	log.Println("==> New Track inserted:", track.Artist, track.Album, track.Track)
}

func getArtistID(artist string) (artistIndex int) {
	for { // try multiple time to get id
		err := Database.QueryRow("SELECT fnAddArtist(?)", strings.TrimSpace(artist)).Scan(&artistIndex)
		if err != nil {
			log.Println("DB Error ArtisT:", err)
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
			log.Println("DB Error AlbuM:", err)
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
		log.Println("DB Error removeTrackDB:", err, id)
	}
}

func TouchTrack(fileName string) {
	_, err := Database.Exec("UPDATE TracK SET TK_LastSeen = CURRENT_TIMESTAMP WHERE TK_FileName = ?", fileName)
	if err != nil {
		log.Println("DB Error touchTrack:", err)
	}
}

func GetCurrentDBTimestamp() (stamp string) {
	err := Database.QueryRow("SELECT CURRENT_TIMESTAMP").Scan(&stamp)
	if err != nil {
		log.Println("DB Error getCurrentDBTimestamp:", err)
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
		log.Println("DB Error getOldTracks:", err, sql, params)
	} else {
		defer rows.Close()
		var back []int
		for rows.Next() {
			var ret int
			err := rows.Scan(&ret)
			if err != nil {
				log.Println("DB Error getOldTracks:", err)
			} else {
				back = append(back, ret)
			}
		}
		return back
	}

	return nil
}
