package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func openDB(conf Configuration) (*sql.DB, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", conf.DBUser, conf.DBPassword, conf.DBServer, conf.DBName))
	if err != nil {
		return nil, err
	}

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func emptyDB() {
	log.Println("=> reset database")
	database.Exec("SET FOREIGN_KEY_CHECKS = 0")
	database.Exec("truncate table ActualPlaying")
	database.Exec("UPDATE DeVice SET DV_LastTKid = NULL")
	database.Exec("truncate table ArtisT")
	database.Exec("truncate table AlbuM")
	database.Exec("truncate table TracK")
	database.Exec("SET FOREIGN_KEY_CHECKS = 1;")
}

func insertTrack(track trackInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	artistIndex := getArtistID(track.artist)
	albumIndex := getAlbumID(track.album, track.albumIndex)

	var trackIndex int
	err := database.QueryRow("SELECT TK_id FROM TracK WHERE TK_FileName = ?", track.fileName).Scan(&trackIndex)
	if err != nil {
		_, err = database.Exec("INSERT INTO TracK (TK_FileName, TK_Name, TK_AT_id, TK_AM_id, TK_Index) VALUES (?,?,?,?,?)", track.fileName, track.track, artistIndex, albumIndex, track.trackIndex)
		if err != nil {
			log.Println("DB Error TracK:", err, track, artistIndex, albumIndex)
		}

		log.Println("==> New Track inserted:", track.artist, track.album, track.track)
	}
}

func updateTrack(track trackInfo) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	artistIndex := getArtistID(track.artist)
	albumIndex := getAlbumID(track.album, track.albumIndex)

	_, err := database.Exec("UPDATE TracK SET TK_Name = ?, TK_AT_id = ?, TK_AM_id = ?, TK_Index = ?, TK_LastSeen = CURRENT_TIMESTAMP WHERE TK_FileName = ?", track.track, artistIndex, albumIndex, track.trackIndex, track.fileName)
	if err != nil {
		log.Println("DB Error TracK:", err, track, artistIndex, albumIndex)
	}

	log.Println("==> New Track inserted:", track.artist, track.album, track.track)
}

func getArtistID(artist string) (artistIndex int) {
	for { // try multiple time to get id
		err := database.QueryRow("SELECT fnAddArtist(?)", strings.TrimSpace(artist)).Scan(&artistIndex)
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
		err := database.QueryRow("SELECT fnAddAlbum(?,?)", strings.TrimSpace(album), index).Scan(&albumIndex)
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

func existsInDB(fileName string) bool {
	var trackIndex int
	err := database.QueryRow("SELECT TK_id FROM TracK WHERE TK_FileName = ?", fileName).Scan(&trackIndex)
	return err == nil
}

func removeTrackDB(id int) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	_, err := database.Exec("CALL spDeleteTrack(?)", id)
	if err != nil {
		log.Println("DB Error removeTrackDB:", err, id)
	}
}

func touchTrack(fileName string) {
	runningJobs <- true
	defer func() { _ = <-runningJobs }()

	_, err := database.Exec("UPDATE TracK SET TK_LastSeen = CURRENT_TIMESTAMP WHERE TK_FileName = ?", fileName)
	if err != nil {
		log.Println("DB Error touchTrack:", err)
	}
}

func getCurrentDBTimestamp() (stamp string) {
	err := database.QueryRow("SELECT CURRENT_TIMESTAMP").Scan(&stamp)
	if err != nil {
		log.Println("DB Error getCurrentDBTimestamp:", err)
	}
	return
}

func getOldTracks(includePath string, exludePaths []string, stamp string) []int {
	sql := "SELECT TK_id FROM TracK WHERE (TK_LastSeen < ?) AND (TK_FileName LIKE ?) "
	var params []interface{}

	params = append(params, stamp)
	params = append(params, includePath+"%")

	for _, exludePath := range exludePaths {
		sql += " AND (TK_FileName NOT LIKE ?) "
		params = append(params, exludePath+"%")
	}

	rows, err := database.Query(sql, params...)
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
