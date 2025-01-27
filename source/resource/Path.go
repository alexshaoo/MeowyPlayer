package resource

import (
	"os"
	"path/filepath"
	"time"

	"meowyplayer.com/utility/assert"
	"meowyplayer.com/utility/json"
)

const (
	albumPath      = "album"
	coverPath      = "cover"
	collectionFile = "collection.json"

	musicPath = "music"
	assetPath = "asset"
)

func CollectionPath() string {
	return filepath.Join(albumPath, collectionFile)
}

func CoverPath(album *Album) string {
	return filepath.Join(albumPath, coverPath, album.Title+".png")
}

func MusicPath(music *Music) string {
	return filepath.Join(musicPath, music.Title)
}

func AssetPath(assetName string) string {
	return filepath.Join(assetPath, assetName)
}

func MakeNecessaryPath() {
	assert.NoErr(os.MkdirAll(filepath.Join(albumPath, coverPath), os.ModePerm), "failed to create cover directory")
	assert.NoErr(os.MkdirAll(filepath.Join(musicPath), os.ModePerm), "failed to create music directory")

	_, err := os.Stat(CollectionPath())
	if os.IsNotExist(err) {
		//create default collection
		assert.NoErr(json.WriteFile(CollectionPath(), &Collection{Date: time.Now(), Albums: nil}), "failed to create default collection")
	} else {
		assert.NoErr(err, "failed to fetch collection path stats")
	}
}
