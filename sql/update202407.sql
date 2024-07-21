ALTER TABLE `ActualPlaying`
	ADD COLUMN `AP_LastSelected` TIMESTAMP NULL DEFAULT NULL AFTER `AP_Playcount`,
	ADD COLUMN `AP_LastPlayed` TIMESTAMP NULL DEFAULT NULL AFTER `AP_LastSelected`,
	ADD COLUMN `AP_Pos` int(10)  NOT NULL DEFAULT '0' AFTER `AP_LastPlayed`;

-- Exportiere Struktur von Tabelle PlaylisT
DROP TABLE IF EXISTS `PlaylisT`;
CREATE TABLE IF NOT EXISTS `PlaylisT` (
  `PT_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `PT_Name` varchar(200) NOT NULL DEFAULT '0',
  PRIMARY KEY (`PT_id`),
  UNIQUE KEY `AM_Name` (`PT_Name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;


-- Exportiere Struktur von Tabelle PlaylistitemS
DROP TABLE IF EXISTS `PlaylistitemS`;
CREATE TABLE IF NOT EXISTS `PlaylistitemS` (
  `PS_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `PS_PT_id` int(10) NOT NULL DEFAULT '0',
  `PS_TK_FileName` varchar(200) NOT NULL DEFAULT '0',
  
  PRIMARY KEY (`PS_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Exportiere Struktur von Funktion radiogo.fnGetNextTrackId
DROP FUNCTION IF EXISTS `fnGetNextTrackId`;
DELIMITER //
CREATE FUNCTION `fnGetNextTrackId`(
    `deviceid` VARCHAR(250),
    `isRandom` INT,
	`currentTrackId` INT(10)

) RETURNS int(11)
BEGIN
    DECLARE TKid INT DEFAULT NULL;
    DECLARE PlayCount INT DEFAULT NULL;
    DECLARE MyPos INT DEFAULT NULL;
    DECLARE NextPos INT DEFAULT NULL;
    SELECT AP_Pos INTO MyPos FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_TK_id = currentTrackId;
	SELECT AP_Pos INTO NextPos FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_Pos = (SELECT AP_Pos FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_TK_id = currentTrackId)+1;
    IF (NextPos > MyPos) THEN
        SELECT AP_TK_id INTO TKid FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_Pos = (SELECT AP_Pos FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_TK_id = currentTrackId)+1;
    ELSEIF (isRandom = 0) THEN
        SELECT TK_id, AP_Playcount INTO TKid, PlayCount FROM ActualPlaying INNER JOIN TracK ON AP_TK_id = TK_id WHERE AP_DV_id = deviceid ORDER BY AP_Playcount, AP_Sort, AP_TK_id LIMIT 1;
    ELSE
    BEGIN	
        DECLARE MinPC INT DEFAULT NULL;
        SELECT MIN(AP_Playcount) INTO MinPC FROM ActualPlaying WHERE AP_DV_id = deviceid;
        SELECT TK_id, AP_Playcount INTO TKid, PlayCount FROM ActualPlaying INNER JOIN TracK ON AP_TK_id = TK_id WHERE AP_Playcount = MinPC AND AP_DV_id = deviceid ORDER BY RAND() LIMIT 1;
    END;
    END IF;

    IF TKid IS NOT NULL THEN
        RETURN TKid;
    ELSE
        RETURN -1;
    END IF;
END//
DELIMITER ;

-- Exportiere Struktur von Funktion radiogo.fnGetPrevTrackID
DROP FUNCTION IF EXISTS `fnGetPrevTrackID`;
DELIMITER //
CREATE FUNCTION `fnGetPrevTrackID`(
	`deviceid` VARCHAR(250),
	`currentTrackId` INT(10)

) RETURNS int(11)
BEGIN
	DECLARE TKid INT DEFAULT NULL;
	SELECT AP_TK_id INTO TKid FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_Pos < (SELECT AP_Pos FROM ActualPlaying WHERE AP_DV_id = deviceid AND AP_TK_id = currentTrackId) order by AP_Pos DESC limit 1;
	
    IF TKid IS NOT NULL THEN
        RETURN TKid;
    ELSE
        RETURN -1;
    END IF;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur radiogo.spMarkTrackPlayed
DROP PROCEDURE `spMarkTackPlayed`;
DROP PROCEDURE IF EXISTS `spMarkTrackPlayed`;
DELIMITER //
CREATE PROCEDURE `spMarkTrackPlayed`(
	IN `deviceid` VARCHAR(250),
	IN `TKid` INT

)
BEGIN
	UPDATE ActualPlaying SET AP_Playcount =  AP_Playcount + 1, AP_LastPlayed = CURRENT_TIMESTAMP WHERE AP_DV_id = deviceid AND AP_TK_id = TKid;
	UPDATE DeVice SET DV_LastTKid = TKid WHERE DV_id = deviceid;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur radiogo.spMarkTrackSelected
DROP PROCEDURE IF EXISTS `spMarkTrackSelected`;
DELIMITER //
CREATE PROCEDURE `spMarkTrackSelected`(
	IN `deviceid` VARCHAR(250),
	IN `TKid` INT

)
BEGIN
	UPDATE ActualPlaying SET AP_LastSelected = CURRENT_TIMESTAMP, AP_Pos=if(AP_Pos=0,(SELECT AP_Pos+1 FROM ActualPlaying WHERE AP_DV_id = deviceid ORDER BY AP_Pos DESC LIMIT 1),AP_Pos) WHERE AP_DV_id = deviceid AND AP_TK_id = TKid;
	UPDATE DeVice SET DV_LastTKid = TKid WHERE DV_id = deviceid;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur spUpdateActualPlaying
DROP PROCEDURE IF EXISTS `spUpdateActualPlaying`;
DELIMITER //
CREATE PROCEDURE `spUpdateActualPlaying`(
	IN `deviceid` VARCHAR(250),
	IN `search` VARCHAR(500)
)
    READS SQL DATA
BEGIN
	IF (SELECT 1 = 1 FROM DeVice WHERE DV_id = deviceid) THEN
	BEGIN
		SET @LikeSearch = CONCAT('%', search, '%');
		DELETE FROM ActualPlaying WHERE AP_DV_id = deviceid;
		
		DROP TABLE IF EXISTS tmppl;
		CREATE TEMPORARY TABLE tmppl (`tmp_id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT, `tmp_TKid` INT(10) UNSIGNED NOT NULL, PRIMARY KEY (`tmp_id`));
		
		INSERT INTO tmppl
	 	SELECT null, TracK.TK_id 
		FROM TracK 
		LEFT JOIN ArtisT ON TK_AT_id = AT_id 
		LEFT JOIN AlbuM ON TK_AM_id = AM_id 
		WHERE TK_Name LIKE @LikeSearch OR TK_Comment LIKE @LikeSearch OR AT_Name LIKE @LikeSearch OR AM_Name LIKE @LikeSearch
		ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;
		
		IF !(SELECT EXISTS (SELECT 1 FROM tmppl)) THEN
		BEGIN
			INSERT INTO tmppl
		 	SELECT null, TracK.TK_id 
			FROM TracK 
			LEFT JOIN ArtisT ON TK_AT_id = AT_id 
			LEFT JOIN AlbuM ON TK_AM_id = AM_id 
			WHERE (TK_Name SOUNDS LIKE search OR TK_Comment SOUNDS LIKE search OR AT_Name SOUNDS LIKE search OR AM_Name SOUNDS LIKE search)
			ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;
		END;
		END IF;
		
		INSERT INTO ActualPlaying
        SELECT deviceid, tmppl.tmp_TKid, tmppl.tmp_id, 0, NULL, NULL, 0 FROM tmppl;
	END;
	END IF;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur spUpdateActualPlaying
DROP PROCEDURE IF EXISTS `spUpdateActualPlayingMusic`;
DELIMITER //
CREATE PROCEDURE `spUpdateActualPlayingMusic`(
        IN `deviceid` VARCHAR(250)
)
    READS SQL DATA
BEGIN
        IF (SELECT 1 = 1 FROM DeVice WHERE DV_id = deviceid) THEN
        BEGIN
                DELETE FROM ActualPlaying WHERE AP_DV_id = deviceid;

                DROP TABLE IF EXISTS tmppl;
                CREATE TEMPORARY TABLE tmppl (`tmp_id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT, `tmp_TKid` INT(10) UNSIGNED NOT NULL, PRIMARY KEY (`tmp_id`));

                INSERT INTO tmppl
                SELECT null, TracK.TK_id
                FROM TracK
                LEFT JOIN ArtisT ON TK_AT_id = AT_id
                LEFT JOIN AlbuM ON TK_AM_id = AM_id
                WHERE TK_FileName LIKE "//musik%"
                ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;

                INSERT INTO ActualPlaying
       			SELECT deviceid, tmppl.tmp_TKid, tmppl.tmp_id, 0, NULL, NULL, 0 FROM tmppl;
        END;
        END IF;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur spUpdateActualPlayingAlbumOrTitle
DROP PROCEDURE IF EXISTS `spUpdateActualPlayingAlbumOrTitle`;
DELIMITER //
CREATE PROCEDURE `spUpdateActualPlayingAlbumOrTitle`(
        IN `deviceid` VARCHAR(250),
        IN `searchAlbumOrTitle` VARCHAR(500),
        IN `searchArtist` VARCHAR(500)
)
    READS SQL DATA
BEGIN
	IF (SELECT 1 = 1 FROM DeVice WHERE DV_id = deviceid) THEN
	BEGIN
		SET @LikeAlbumOrTitle = CONCAT('%', searchAlbumOrTitle, '%');
		SET @LikeArtist = CONCAT('%', searchArtist, '%');
		DELETE FROM ActualPlaying WHERE AP_DV_id = deviceid;

		DROP TABLE IF EXISTS tmppl;
		CREATE TEMPORARY TABLE tmppl (`tmp_id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT, `tmp_TKid` INT(10) UNSIGNED NOT NULL, PRIMARY KEY (`tmp_id`));

		INSERT INTO tmppl
		SELECT null, TracK.TK_id
		FROM TracK
		LEFT JOIN ArtisT ON TK_AT_id = AT_id
		LEFT JOIN AlbuM ON TK_AM_id = AM_id
		WHERE AT_Name LIKE @LikeArtist AND (TK_Name LIKE @LikeAlbumOrTitle OR TK_Comment LIKE @LikeAlbumOrTitle OR AM_Name LIKE @LikeAlbumOrTitle)
		ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;

		IF !(SELECT EXISTS (SELECT 1 FROM tmppl)) THEN
		BEGIN
			INSERT INTO tmppl
			SELECT null, TracK.TK_id
			FROM TracK
			LEFT JOIN ArtisT ON TK_AT_id = AT_id
			LEFT JOIN AlbuM ON TK_AM_id = AM_id
			WHERE AT_Name SOUNDS LIKE searchArtist AND (TK_Name SOUNDS LIKE searchAlbumOrTitle OR TK_Comment SOUNDS LIKE searchAlbumOrTitle OR AM_Name SOUNDS LIKE searchAlbumOrTitle)
			ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;
		END;
		END IF;

		INSERT INTO ActualPlaying
        SELECT deviceid, tmppl.tmp_TKid, tmppl.tmp_id, 0, NULL, NULL, 0 FROM tmppl;
	END;
	END IF;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur spUpdateActualPlaying
DROP PROCEDURE IF EXISTS `spUpdateActualPlayingMusic`;
DELIMITER //
CREATE PROCEDURE `spUpdateActualPlayingMusic`(
        IN `deviceid` VARCHAR(250)
)
    READS SQL DATA
BEGIN
        IF (SELECT 1 = 1 FROM DeVice WHERE DV_id = deviceid) THEN
        BEGIN
                DELETE FROM ActualPlaying WHERE AP_DV_id = deviceid;

                DROP TABLE IF EXISTS tmppl;
                CREATE TEMPORARY TABLE tmppl (`tmp_id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT, `tmp_TKid` INT(10) UNSIGNED NOT NULL, PRIMARY KEY (`tmp_id`));

                INSERT INTO tmppl
                SELECT null, TracK.TK_id
                FROM TracK
                LEFT JOIN ArtisT ON TK_AT_id = AT_id
                LEFT JOIN AlbuM ON TK_AM_id = AM_id
                WHERE TK_FileName LIKE "//musik%" 
                ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;

                INSERT INTO ActualPlaying
        		SELECT deviceid, tmppl.tmp_TKid, tmppl.tmp_id, 0, NULL, NULL, 0 FROM tmppl;
        END;
        END IF;
END//
DELIMITER ;

-- Exportiere Struktur von Prozedur spUpdateActualPlayList
DROP PROCEDURE IF EXISTS `spUpdateActualPlayList`;
DELIMITER //
CREATE PROCEDURE `spUpdateActualPlayList`(
        IN `deviceid` VARCHAR(250),
        IN `SearchString` VARCHAR(500)
)
    READS SQL DATA
BEGIN
	IF (SELECT 1 = 1 FROM DeVice WHERE DV_id = deviceid) THEN
	BEGIN
		SET @PlayList = CONCAT('%', SearchString, '%');
		DELETE FROM ActualPlaying WHERE AP_DV_id = deviceid;

		DROP TABLE IF EXISTS tmppl;
		CREATE TEMPORARY TABLE tmppl (`tmp_id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT, `tmp_TKid` INT(10) UNSIGNED NOT NULL, PRIMARY KEY (`tmp_id`));

		INSERT INTO tmppl
		SELECT null, TracK.TK_id
		FROM PlaylistitemS
		LEFT JOIN PlaylisT on PT_id = PS_PT_id
		LEFT JOIN TracK ON PS_TK_FileName = TK_FileName
		LEFT JOIN ArtisT ON TK_AT_id = AT_id
		LEFT JOIN AlbuM ON TK_AM_id = AM_id
		WHERE PT_Name SOUNDS LIKE @PlayList;

		INSERT INTO ActualPlaying
        SELECT deviceid, tmppl.tmp_TKid, tmppl.tmp_id, 0, NULL, NULL, 0 FROM tmppl;
	END;
	END IF;
END//
DELIMITER ;

