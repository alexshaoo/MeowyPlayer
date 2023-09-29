package client

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"golang.org/x/exp/slices"
	"meowyplayer.com/source/player"
	"meowyplayer.com/source/resource"
	"meowyplayer.com/source/utility"
)

var collectionData utility.Data[player.Collection]
var albumData utility.Data[player.Album]
var playListData utility.Data[player.PlayList]

// the album pointer parameter may refer to a temporary object from the view list
// we need the original one from the collection
func getSourceAlbum(album *player.Album) *player.Album {
	index := slices.IndexFunc(collectionData.Get().Albums, func(a player.Album) bool { return a.Title == album.Title })
	return &collectionData.Get().Albums[index]
}

func reloadCollection() error {
	if err := utility.WriteJson(resource.CollectionPath(), collectionData.Get()); err != nil {
		return err
	}
	collection, err := LoadFromLocalCollection()
	if err != nil {
		return err
	}
	collectionData.Set(&collection)
	return nil
}

func reloadAlbum() error {
	albumData.Set(getSourceAlbum(albumData.Get()))
	return nil
}

func LoadFromLocalCollection() (player.Collection, error) {
	inUse := player.Collection{}
	if err := utility.ReadJson(resource.CollectionPath(), &inUse); err != nil {
		return inUse, err
	}

	for i := range inUse.Albums {
		album := &inUse.Albums[i]
		album.Cover = resource.GetCover(album)

		//read music file size
		for j := range album.MusicList {
			music := &album.MusicList[j]
			fileInfo, err := os.Stat(resource.MusicPath(music))
			utility.ShouldNil(err)
			music.FileSize = fileInfo.Size()
		}
	}

	return inUse, nil
}

func GetCurrentCollection() *utility.Data[player.Collection] {
	return &collectionData
}

func GetCurrentAlbum() *utility.Data[player.Album] {
	return &albumData
}

func GetCurrentPlayList() *utility.Data[player.PlayList] {
	return &playListData
}

func AddAlbum() error {
	inUse := collectionData.Get()

	//generate title
	title := ""
	for i := 0; i < math.MaxInt; i++ {
		title = fmt.Sprintf("Album (%v)", i)
		if !slices.ContainsFunc(inUse.Albums, func(a player.Album) bool { return a.Title == title }) {
			break
		}
	}

	//generate album
	album := player.Album{Date: time.Now(), Title: title}
	inUse.Albums = append(inUse.Albums, album)

	//generate album cover
	iconColor := color.NRGBA{uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32()), uint8(rand.Uint32())}
	iconImage := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	iconImage.SetNRGBA(0, 0, iconColor)
	imageData := bytes.Buffer{}
	if err := png.Encode(&imageData, iconImage); err != nil {
		return err
	}
	if err := os.WriteFile(resource.CoverPath(&album), imageData.Bytes(), os.ModePerm); err != nil {
		return err
	}

	return reloadCollection()
}

func AddMusic(musicInfo fyne.URIReadCloser) error {
	//add to the source album
	music := player.Music{Date: time.Now(), Title: musicInfo.URI().Name()}
	album := getSourceAlbum(albumData.Get())
	album.MusicList = append(album.MusicList, music)

	//copy the music file to the music repo
	musicFile, err := os.ReadFile(musicInfo.URI().Path())
	if err != nil {
		return err
	}
	if err = os.WriteFile(resource.MusicPath(&music), musicFile, os.ModePerm); err != nil {
		return err
	}
	if err := reloadCollection(); err != nil {
		return err
	}
	return reloadAlbum()
}

func DeleteAlbum(album *player.Album) error {
	collection := collectionData.Get()
	index := slices.IndexFunc(collection.Albums, func(a player.Album) bool { return a.Title == album.Title })
	last := len(collection.Albums) - 1

	//remove album icon
	if err := os.Remove(resource.CoverPath(album)); err != nil && !os.IsNotExist(err) {
		return err
	}

	//pop from the collection
	collection.Albums[index] = collection.Albums[last]
	collection.Albums = collection.Albums[:last]
	return reloadCollection()
}

func DeleteMusic(music *player.Music) error {
	album := getSourceAlbum(albumData.Get())
	index := slices.IndexFunc(album.MusicList, func(m player.Music) bool { return m.SimpleTitle() == music.SimpleTitle() })
	last := len(album.MusicList) - 1

	//pop form the album
	album.MusicList[index] = album.MusicList[last]
	album.MusicList = album.MusicList[:last]

	if err := reloadCollection(); err != nil {
		return err
	}
	return reloadAlbum()
}

func UpdateAlbumTitle(album *player.Album, title string) error {
	if slices.ContainsFunc(collectionData.Get().Albums, func(a player.Album) bool { return a.Title == title }) {
		return fmt.Errorf("album \"%v\" already exists", title)
	}

	//update timestamp
	collectionData.Get().Date = time.Now()
	source := getSourceAlbum(album)
	source.Date = time.Now()

	//rename the album cover
	oldPath := resource.CoverPath(source)
	source.Title = title
	if err := os.Rename(oldPath, resource.CoverPath(source)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return reloadCollection()
}

func UpdateAlbumCover(album *player.Album, iconPath string) error {
	album = getSourceAlbum(album)

	//update timestamp
	album.Date = time.Now()
	collectionData.Get().Date = time.Now()

	//update cover image
	icon, err := os.ReadFile(iconPath)
	if err != nil {
		return err
	}
	if err = os.WriteFile(resource.CoverPath(album), icon, os.ModePerm); err != nil {
		return err
	}
	return reloadCollection()
}