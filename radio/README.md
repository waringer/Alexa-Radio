# radio

**--- Deutsch ---**

radio.go ist der eigentliche Alexa-Skill

Parameter:

* -v
    * Anzeigen der Version, skill wird nicht gestartet!
* -c conf.file
    * Definieren einer alternativen Konfigurationsdatei, Default:radio.conf, wird die Datei nicht gefunden wird sie mit Beispieldaten erstellt.

### Funktionsumfang
Der Skill versteht die folgenden eigenen Kommandos:

* Musik
  * Setzt die Wiedergabe der aktuellen Playlist für das aktuelle Gerät fort
* Suche [Suchbegriff]
  * Sucht in der Bibliothek alle Künstler, Alben und Titel die auf den Suchbegriff passen und erstellt damit eine neue Playlist.
  * Die Playlist wird sofort abgespielt
* Was läuft gerade
* Was ist das
  * Beides gibt zurück was gerade abgespielt wird oder zuletzt abgespielt wurde

Daneben versteht der Skill auch die folgenden Standard Kommandos:

* Stop
* Nächster
* Weiter
* Pause
* Shuffle an/aus
* Loop an/aus


**--- English ---**

soon
