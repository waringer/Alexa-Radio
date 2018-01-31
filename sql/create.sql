/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

-- Exportiere Struktur von Tabelle radiogo.AlbuM
DROP TABLE IF EXISTS `AlbuM`;
CREATE TABLE IF NOT EXISTS `AlbuM` (
  `AM_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `AM_Name` varchar(200) NOT NULL DEFAULT '0',
  `AM_Index` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`AM_id`),
  UNIQUE KEY `AM_Name` (`AM_Name`)
) ENGINE=InnoDB AUTO_INCREMENT=20 DEFAULT CHARSET=utf8;

-- Exportiere Struktur von Tabelle radiogo.ArtisT
DROP TABLE IF EXISTS `ArtisT`;
CREATE TABLE IF NOT EXISTS `ArtisT` (
  `AT_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `AT_Name` varchar(200) NOT NULL DEFAULT '0',
  PRIMARY KEY (`AT_id`),
  UNIQUE KEY `AT_Name` (`AT_Name`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8;

-- Exportiere Struktur von Funktion radiogo.fnAddAlbum
DROP FUNCTION IF EXISTS `fnAddAlbum`;
DELIMITER //
CREATE DEFINER=`radiogo`@`%` FUNCTION `fnAddAlbum`(
	`Name` VARCHAR(500),
	`AMIndex` INT
) RETURNS int(11)
    MODIFIES SQL DATA
BEGIN
	DECLARE AMid INT;
	SET @AMid = -1;
	
	SELECT AM_id INTO @AMid FROM AlbuM WHERE AM_Name = Name;
	
	IF @AMid = -1 THEN
		INSERT IGNORE INTO AlbuM (AM_Name, AM_Index) VALUES (Name, AMIndex);
		SELECT AM_id INTO @AMid FROM AlbuM WHERE AM_Name = Name;
	END IF;

	RETURN @AMid;
END//
DELIMITER ;

-- Exportiere Struktur von Funktion radiogo.fnAddArtist
DROP FUNCTION IF EXISTS `fnAddArtist`;
DELIMITER //
CREATE DEFINER=`radiogo`@`%` FUNCTION `fnAddArtist`(
	`Name` VARCHAR(500)
) RETURNS int(11)
    MODIFIES SQL DATA
BEGIN
	DECLARE ATid INT;
	SET @ATid = -1;

	SELECT AT_id INTO @ATid FROM ArtisT WHERE AT_Name = Name;
	
	IF @ATid = -1 THEN
		INSERT IGNORE INTO ArtisT (AT_Name) VALUES (Name);
		SELECT AT_id INTO @ATid FROM ArtisT WHERE AT_Name = Name;
	END IF;

	RETURN @ATid;
END//
DELIMITER ;

-- Exportiere Struktur von Tabelle radiogo.TracK
DROP TABLE IF EXISTS `TracK`;
CREATE TABLE IF NOT EXISTS `TracK` (
  `TK_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `TK_AT_id` int(10) unsigned NOT NULL,
  `TK_AM_id` int(10) unsigned NOT NULL,
  `TK_Index` int(10) unsigned NOT NULL DEFAULT '0',
  `TK_Name` varchar(200) NOT NULL,
  `TK_FileName` varchar(2000) NOT NULL,
  PRIMARY KEY (`TK_id`),
  KEY `Artist` (`TK_AT_id`),
  KEY `Album` (`TK_AM_id`),
  KEY `TK_FileName` (`TK_FileName`(255)),
  CONSTRAINT `Album` FOREIGN KEY (`TK_AM_id`) REFERENCES `AlbuM` (`AM_id`),
  CONSTRAINT `Artist` FOREIGN KEY (`TK_AT_id`) REFERENCES `ArtisT` (`AT_id`)
) ENGINE=InnoDB AUTO_INCREMENT=317 DEFAULT CHARSET=utf8;

/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
