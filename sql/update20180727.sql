/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

-- Exportiere Struktur von Funktion radiogo.fnGetNextTrackId
DELIMITER //
CREATE FUNCTION `fnGetNextTrackId`(
	`deviceid` VARCHAR(250),
	`isRandom` INT

) RETURNS int(11)
BEGIN
	DECLARE TKid INT DEFAULT NULL;
	DECLARE PlayCount INT DEFAULT NULL;
	
	IF (isRandom = 0) THEN
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

-- Exportiere Struktur von Prozedur radiogo.spMarkTackPlayed
DELIMITER //
CREATE PROCEDURE `spMarkTackPlayed`(
	IN `deviceid` VARCHAR(250),
	IN `TKid` INT

)
BEGIN
	UPDATE ActualPlaying SET AP_Playcount =  AP_Playcount + 1 WHERE AP_DV_id = deviceid AND AP_TK_id = TKid;
	UPDATE DeVice SET DV_LastTKid = TKid WHERE DV_id = deviceid;
END//
DELIMITER ;

/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
