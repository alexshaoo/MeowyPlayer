package player

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2/canvas"
	"meowyplayer.com/source/pattern"
	"meowyplayer.com/source/resource"
)

var state *State

func init() {
	state = NewState()
}

func GetState() *State {
	return state
}

type State struct {
	album                Album
	musics               []Music
	onReadAlbumsFromDisk pattern.OneArgSubject[[]Album]
	onSelectAlbum        pattern.OneArgSubject[Album]
	onReadMusicsDisk     pattern.OneArgSubject[[]Music]
	onSelectMusic        pattern.ThreeArgSubject[Album, []Music, Music]
}

func NewState() *State {
	return &State{}
}

func (state *State) OnReadAlbumsFromDisk() *pattern.OneArgSubject[[]Album] {
	return &state.onReadAlbumsFromDisk
}

func (state *State) OnSelectAlbum() *pattern.OneArgSubject[Album] {
	return &state.onSelectAlbum
}

func (state *State) OnReadMusicFromDisk() *pattern.OneArgSubject[[]Music] {
	return &state.onReadMusicsDisk
}

func (state *State) OnSelectMusic() *pattern.ThreeArgSubject[Album, []Music, Music] {
	return &state.onSelectMusic
}

func (state *State) SetSelectedAlbum(album Album) {
	state.onSelectAlbum.NotifyAll(album)
	if state.album != album {
		state.album = album
		state.musics = ReadMusicFromDisk(album)
		state.onReadMusicsDisk.NotifyAll(state.musics)
	}
}

func (state *State) SetSelectedMusic(music Music) {
	state.onSelectMusic.NotifyAll(state.album, state.musics, music)
}

func ReadAlbumsFromDisk() []Album {
	directories, err := os.ReadDir(resource.GetAlbumFolderPath())
	if err != nil {
		log.Fatal(err)
	}

	const bufferSize = 32 * 1024 //arbitrary magic number
	buffer := make([]byte, bufferSize)
	albums := []Album{}

	for _, directory := range directories {
		if directory.IsDir() {

			//read album config
			config, err := os.Open(resource.GetAlbumConfigPath(directory.Name()))
			if err != nil {
				log.Fatal(err)
			}
			defer config.Close()

			//read last modified date
			info, err := config.Stat()
			if err != nil {
				log.Fatal(err)
			}

			//read number of music
			bytesRead, err := config.Read(buffer)
			if err != nil && err != io.EOF {
				log.Fatal(err)
			}
			musicNumber := bytes.Count(buffer[:bytesRead], []byte{'\n'})

			//get album cover
			albumCover := canvas.NewImageFromFile(resource.GetAlbumIconPath(directory.Name()))

			albums = append(albums, Album{directory.Name(), info.ModTime(), musicNumber, albumCover})
		}
	}
	return albums
}

func ReadMusicFromDisk(album Album) []Music {
	config, err := os.Open(resource.GetAlbumConfigPath(album.title))
	if err != nil {
		log.Fatal(err)
	}
	defer config.Close()

	//read music name from config
	music := []Music{}
	scanner := bufio.NewScanner(config)
	for scanner.Scan() {

		//open music file
		info, err := os.Stat(resource.GetMusicPath(scanner.Text()))
		if err != nil {
			log.Fatal(err)
		}

		music = append(music, Music{scanner.Text(), estimateDuration(info), info.ModTime()})
	}

	if scanner.Err() != nil {
		log.Fatal(scanner.Err())
	}

	return music
}

// a very rough estimation of the music duration in nanoseconds
func estimateDuration(musicFileInfo fs.FileInfo) time.Duration {
	return time.Duration(musicFileInfo.Size() * MAGIC_RATIO / (AUDIO_BIT_DEPTH * NUM_OF_CHANNELS * SAMPLING_RATE))
}
