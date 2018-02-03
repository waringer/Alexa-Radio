# Alexa-Radio

Amazon Echo/Alexa Skill zum abspielen von Musik von einem NAS / Webserver
**Skill ist noch WIP und hat aktull nur eine sehr Rudimäntäre Funktion.**

## Konzeption
Die Musik liegt auf einem NAS und wird über einen Webserver bereitgestellt.
Dem Echo wird über den Skill der Link zur der Musik auf dem Webserver geliefert, dieser Spielt die Musik ab.

	(NAS) <- NFS -> (Interner Webserver) <- HTTP -> (Echo)

Benötigt wird:

* Musikdateien in mp3, flac (nur wenn der interne Webserver das OnTheFly in mp3 umwandeln kann - siehe utils), ogg oder m4a Format
* Ein "interner" Webserver (z.B. nginx) der auf die Musikdateien zugreifen kann
* Ein "externer" Webserver der den Skill offiziell Hostet und von Amazon angesprochen wird (muss eins gültiges Zertifikat für HTTPS haben!)
* Ein Server auf dem der Skill läuft - kann z.B. auch der Server mit dem externen Webserver sein
* Ein MySQL oder MariaDB Server - der Skill verwaltet darin die Playlisten

Der externe und der interne Webserver können auch identisch sein, man sollte dann nur dafür sorgen das die Musikdateien nicht für jeden im Netz abrufbar sind.

Die Bilder für den Skill sind von "ypf" (<http://ypf.deviantart.com/art/icon-for-transformers-61630983>) (<https://www.iconfinder.com/icons/31262/music_my_icon>)

**--- English ---**

Skill to play music from a webserver - **!!Early Work In Progrss!!**
Detailed english description follows later, sorry folks.

Images from "ypf" (<http://ypf.deviantart.com/art/icon-for-transformers-61630983>) (<https://www.iconfinder.com/icons/31262/music_my_icon>)
