# music bot for memes

Flags:

```cmd
--token="Your Discord bot token"
--guild="Guild ID"
--app="Application ID"
--yt="YouTube API Key"
```

## Features

* Youtube links (/add url)
* Youtube playlists (/add url) with concurrent downloads
* Specified timestamp for videos (e.g. ?t=20) (/add url)
* Direct video/audio uploads from discord attachments (/add file)
* Automatically join voice and play (/add url)
* Pause and unpause with the same command (/pause)
* Display current song queue (/queue)
* --cookies support for yt-dlp for age restricted videos
 * put cookies.txt in the same directory as the bot
* Queue manipulation:
  * Skip (/skip)
  * Clear (/clear)
  * Shuffle (/shuffle)
  * Remove (/remove index)
