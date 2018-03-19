# config

**--- Deutsch ---**

Die Konfigurationsdatei ist im Json Format und enthält die folgenden Parameter (Casesensitive):

* bindingIP
    * IP auf den der Skill auf Anfragen von Amazon wartet.
    * *Wird nur für den Skill verwendet*
* bindingPort
    * TCP Port auf den der Skill auf Anfragen wartet. Default:3081
    * *Wird nur für den Skill verwendet*
* amazonAppID
    * Die Amazon AppID
    * *Wird nur für den Skill verwendet*
* pidFile
    * Datei in der die ProzessID des Skills geschrieben wird. Default:/var/run/alexa_radio.pid
    * *Wird nur für den Skill verwendet*
* streamURL
    * Die Basis URL unter dem die Musik Dateien abrufbar sind.
* dbUser
    * Username für die Verbindung mit dem Datenbank Server.
* dbPassword
    * Passwort für die Verbindung mit dem Datenbank Server.
* dbName
    * Datenbankname auf dem Datenbank Server in dem die Daten verwaltet werden.
* dbServer
    * IP oder Name des Datenbank Servers. Default:localhost
* scannerConfiguration
    * Liste der Konfigurationen der Verzeichnisse in dem der Scanner nach Dateien sucht.
    * **Genaue Beschreibung folgt**
    * *Wird nur für den Scanner verwendet*

Ein Element der Scannerconfiguration enthält die folgenden Parameter:

* useTags
    * "*true*" oder "*false*"
    * Default:*true*
    * Bei *true* versuchen ID3 Tags aus den Datein zu lesen, wenn *false* oder keine Tags gelesen werden können wird zur ermittlung der Daten der Parameter "*tagExtractors*" verwendet.
* fileAccessMode
    * "*local*" oder "*nfs*"
    * Default:*local*
    * Gibt an ob für den Zugriff auf die Dateien NFS verwendet werden soll oder das lokale Dateisystem.
* removeNoLongerExisting
    * "*true*" oder "*false*"
    * Default:*false*
    * Wenn *true*, bei Dateien die nicht mehr existieren den ensprechenden Datenbank Eintrag löschen.
* localBasePath
    * Basis Pfad in dem gesucht wird wenn "*fileAccessMode*" "*local*" ist.
* nfsServer
    * IP oder Name des NFS Servers der Verwendet wird wenn "*fileAccessMode*" "*nfs*" ist.
    * Es wird NFS in der Version 3 verwendet.
* nfsShare
    * Name des NFS Shares der Verwendet wird wenn "*fileAccessMode*" "*nfs*" ist.
* validExtensions
    * Liste der Dateiendungen die als gültige Musikdateien angesehen werden.
    * Default:
        * "*.mp3":true*
        * "*.flac":true*
        * "*.ogg":true*
* pathIncludes
* pathExcludes
* tagExtractors

**--- English ---**

soon
