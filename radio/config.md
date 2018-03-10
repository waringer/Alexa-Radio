# config

**--- Deutsch ---**

Die Konfigurationsdatei ist im Json Format und enthält die folgenden Parameter (Casesensitive):
* bindingIP<br/>
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
    * Konfiguration der Verzeichnisse in dem der Scanner nach Dateien sucht.
    * **Genaue Beschreibung folgt**
    * *Wird nur für den Scanner verwendet*

**--- English ---**

soon
