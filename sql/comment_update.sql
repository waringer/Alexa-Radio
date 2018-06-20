/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

ALTER TABLE `TracK`
	ADD COLUMN `TK_Comment` VARCHAR(2000) NOT NULL DEFAULT '' AFTER `TK_Name`;

-- Exportiere Struktur von Prozedur radiogo.spUpdateActualPlaying
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
		SET search = CONCAT('%', search, '%');
		DELETE FROM ActualPlaying WHERE AP_DV_id = deviceid;
		
		CREATE OR REPLACE TEMPORARY TABLE tmppl (`tmp_id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT, `tmp_TKid` INT(10) UNSIGNED NOT NULL, PRIMARY KEY (`tmp_id`));
		
		INSERT INTO tmppl
	 	SELECT null, TracK.TK_id 
		FROM TracK 
		LEFT JOIN ArtisT ON TK_AT_id = AT_id 
		LEFT JOIN AlbuM ON TK_AM_id = AM_id 
		WHERE TK_Name LIKE search OR TK_Comment LIKE search OR AT_Name LIKE search OR AM_Name LIKE search 
		ORDER BY AT_id, AM_Index, AM_id, TK_Index, TK_id;
		
		INSERT INTO ActualPlaying
	 	SELECT deviceid, tmppl.tmp_TKid, tmppl.tmp_id, 0 FROM tmppl;
	END;
	END IF;
END//
DELIMITER ;

/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
