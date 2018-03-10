# Alexa-Radio

**--- Deutsch ---**

Amazon Echo/Alexa Skill zum abspielen von Musik von einem NAS / Webserver
**Der Skill ist noch in der Entwicklung und hat noch nicht alle Funktionen.**

## Konzeption
Die Musik liegt auf einem NAS und wird über einen Webserver bereitgestellt.
Dem Echo wird über den Skill der Link zur der Musik auf dem Webserver geliefert, dieser Spielt die Musik ab.

	(NAS) <- NFS -> (Interner Webserver) <- HTTP -> (Echo)

Benötigt wird:

* Musikdateien in mp3, flac (nur wenn der interne Webserver das OnTheFly in mp3 umwandeln kann - siehe utils), ogg oder m4a Format
* Ein "interner" Webserver (z.B. nginx) der auf die Musikdateien zugreifen kann - ist im selben Netz wie der Echo
* Ein "externer" Webserver der den Skill offiziell Hostet und von Amazon angesprochen wird (muss eins gültiges Zertifikat für HTTPS haben!)
* Ein Server auf dem der Skill läuft - kann z.B. auch der Server mit dem externen Webserver sein
* Ein MySQL oder MariaDB Server - der Skill verwaltet darin die Playlisten

Der externe und der interne Webserver können auch identisch sein, man sollte dann nur dafür sorgen das die Musikdateien nicht für jeden im Netz abrufbar sind.

## Beschreibung
Wenn ein Titel abgespielt werden soll, liefert der Skill einen Link auf den externen Webserver (z.B. <https://extern.alexaendpoint.sample/music/sample.mp3>). Der externe Webserver sendet einen Redirect auf den internen Webserver (z.B. <http://intern.webserver.sample:88/sample.mp3>) und der Echo spielt die Musik vom internen Webserver ab.

Der Umweg über den externen Webserver ist leider nötig da Amazon keine Rückgaben akzeptiert die auf nicht öffentliche Resourcen zeigen. Amazon prüft scheinbar die IP Adresse des Webservers auf dem die Musik liegt und blockt aktiv alles was in einem Privatem IP Netz liegt. Der Echo selber prüft aber nicht und folgt einfach dem Redirect.

Das Projekt besteht aktuell aus 2 Teilen:
* radio.go - dem eigentlichen Skill
* scanner.go - Tool zum katalogisieren der Musikdateien in der Datenbank


Die Bilder für den Skill sind von "ypf" (<http://ypf.deviantart.com/art/icon-for-transformers-61630983>) (<https://www.iconfinder.com/icons/31262/music_my_icon>)

**--- English ---**

Amazon Echo/Alexa Skill to play music from a NAS / webserver
**Skill is still in development and not is not feature complete.**

## Conception
The music are provided from a webserver (The files resides on a NAS).
The Echo became the link to the webserver with the music from the skill. The Echo plays the music.

	(NAS) <- NFS -> (Internal Webserver) <- HTTP -> (Echo)

Required:

* The music files in mp3, ogg, m4a or flac (only if the internal webserver can convert it on the fly to mp3 - see utils) format
* A "internal" webserver (such as nginx) that have access to the music files on the NAS - is in the same network as the Echo
* A "external" webserver, endpoint for the skill, requires a valid certificate for HTTPS
* A Server on which thw skill runns - should be the same as the external webserver
* A MySQL or MariaDB server - used by the skill to store the playlists

The internal and the external webserver may be the same, but then look that not everyone will have access to the music files!

## Description
To play a track the skill provide a link to the external webserver (such as <https://extern.alexaendpoint.sample/music/sample.mp3>). The external webserver then sends a redirect to the internal webserver (such as <http://intern.webserver.sample:88/sample.mp3>) and the Echo plays the music from the internal webserver.

The indirection over the external webserver is required because Amazon rejects links to private resources. It looks like Amazon checks if the IP of the webserver is in a private network and activly blocks such resonses. The Echo itself don't make any checks and follow the redirects.

The project actual have 2 parts:
* radio.go - the skill itself
* scanner.go - tool to catalog the music files into the database


Images from "ypf" (<http://ypf.deviantart.com/art/icon-for-transformers-61630983>) (<https://www.iconfinder.com/icons/31262/music_my_icon>)
