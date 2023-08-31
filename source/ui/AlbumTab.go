package ui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/exp/slices"
	"meowyplayer.com/source/player"
	"meowyplayer.com/source/resource"
	"meowyplayer.com/source/ui/cbinding"
	"meowyplayer.com/source/utility"
)

func newAlbumTab() *container.TabItem {
	const (
		albumTabTitle      = "Album"
		albumTabIconName   = "album_tab.png"
		albumAdderIconName = "album_adder.png"
	)

	// album views
	data := cbinding.NewConfigList()
	resource.GetCurrentConfig().Attach(data)
	view := newAlbumView(data)

	//search bar
	searchBar := widget.NewEntry()
	searchBar.OnChanged = func(title string) {
		data.SetFilter(makeFilter(title))
		view.ScrollToTop()
	}

	albumAdderButton := widget.NewButtonWithIcon("", resource.GetTexture(albumAdderIconName), func() {})
	albumAdderButton.Importance = widget.LowImportance

	//sort by title button
	reverseTitle := false
	sortByTitleButton := widget.NewButton("Title", func() {
		reverseTitle = !reverseTitle
		data.SetSorter(makeTitleSorter(reverseTitle))
	})
	sortByTitleButton.Importance = widget.LowImportance

	//sort by date button
	reverseDate := true
	sortByDateButton := widget.NewButton("Date", func() {
		reverseDate = !reverseDate
		data.SetSorter(makeDateSorter(reverseDate))
	})
	sortByDateButton.Importance = widget.LowImportance
	sortByDateButton.OnTapped()

	border := container.NewBorder(
		container.NewBorder(
			nil,
			container.NewGridWithRows(1, sortByTitleButton, sortByDateButton),
			nil,
			albumAdderButton,
			searchBar,
		),
		nil,
		nil,
		nil,
		view,
	)
	return container.NewTabItemWithIcon(albumTabTitle, resource.GetTexture(albumTabIconName), border)
}

func newAlbumView(data binding.DataList) *widget.List {
	const (
		albumCoverIconName = "default.png"
	)
	albumCoverIconSize := fyne.NewSize(128.0, 128.0)
	albumCoverIcon := resource.GetTexture(albumCoverIconName)

	view := widget.NewListWithData(
		data,
		func() fyne.CanvasObject {
			setting := widget.NewButton(">", func() {})
			setting.Importance = widget.LowImportance
			cover := widget.NewCard("", "", nil)
			cover.Image = canvas.NewImageFromResource(albumCoverIcon)
			intro := widget.NewLabel("")
			return container.NewBorder(nil, nil, container.NewBorder(nil, nil, setting, nil, cover), nil, intro)
		},
		func(item binding.DataItem, canvasObject fyne.CanvasObject) {
			data, err := item.(binding.Untyped).Get()
			utility.MustOk(err)
			album := data.(*player.Album)

			objects0 := canvasObject.(*fyne.Container).Objects
			intro := objects0[0].(*widget.Label)

			//optionally update
			if description := album.Description(); intro.Text != description {
				intro.Text = description

				objects1 := objects0[1].(*fyne.Container).Objects

				//update album cover
				cover := objects1[0].(*widget.Card)
				utility.MustNotNil(cover)
				cover.Image = album.Cover
				cover.Image.SetMinSize(albumCoverIconSize)

				//update setting functionality
				setting := objects1[1].(*widget.Button)
				utility.MustNotNil(setting)
				setting.OnTapped = func() {
					newAlbumMenu(fyne.CurrentApp().Driver().CanvasForObject(canvasObject), album).ShowAtPosition(
						fyne.CurrentApp().Driver().AbsolutePositionForObject(setting))
				}

				canvasObject.Refresh()
			}
		})

	//select and load album
	view.OnSelected = func(id widget.ListItemID) {
		view.Unselect(id)
	}

	return view
}

func newAlbumMenu(canvas fyne.Canvas, album *player.Album) *widget.PopUpMenu {
	rename := fyne.NewMenuItem("Rename", makeRenameDialog(album))
	cover := fyne.NewMenuItem("Cover", makeCoverDialog(album))
	delete := fyne.NewMenuItem("Delete", makeDeleteDialog(album))
	return widget.NewPopUpMenu(fyne.NewMenu("", rename, cover, delete), canvas)
}

func makeRenameDialog(album *player.Album) func() {
	entry := widget.NewEntry()
	return func() {
		dialog.ShowCustomConfirm("Enter title:", "Confirm", "Cancel", entry, func(rename bool) {
			if rename {
				log.Printf("rename %v to %v\n", album.Title, entry.Text)
				showErrorIfAny(updateAlbumTitle(album, entry.Text))
			}
		}, getMainWindow())
	}
}

func makeCoverDialog(album *player.Album) func() {
	return func() {
		fileOpenDialog := dialog.NewFileOpen(func(result fyne.URIReadCloser, err error) {
			if err != nil {
				showErrorIfAny(err)
			} else if result != nil {
				log.Printf("update %v's cover: %v\n", album.Title, result.URI().Path())
				showErrorIfAny(updateAlbumCover(album, result.URI().Path()))
			}
		}, getMainWindow())
		fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", "jpeg", ".bmp"}))
		fileOpenDialog.SetConfirmText("Upload")
		fileOpenDialog.Show()
	}
}

func makeDeleteDialog(album *player.Album) func() {
	return func() {
		dialog.ShowConfirm("", fmt.Sprintf("Do you want to delete %v?", album.Title), func(delete bool) {
			if delete {
				log.Printf("delete %v\n", album.Title)
				showErrorIfAny(deleteAlbum(album))
			}
		}, getMainWindow())
	}
}

func updateAlbumTitle(album *player.Album, title string) error {
	config := resource.GetCurrentConfig()
	config.Date = time.Now()

	//rename album
	oldPath := resource.GetIconPath(album)
	album.Title = title
	album.Date = time.Now()

	//rename album icon if name changed
	if album.Title != title {
		if err := resource.SetIcon(album, oldPath); err != nil {
			return err
		}
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return resource.ReloadCurrentConfig()
}

func updateAlbumCover(album *player.Album, iconPath string) error {
	config := resource.GetCurrentConfig()
	config.Date = time.Now()
	album.Date = time.Now() //update description -> update album view
	if err := resource.SetIcon(album, iconPath); err != nil {
		return err
	}
	return resource.ReloadCurrentConfig()
}

func deleteAlbum(album *player.Album) error {
	config := resource.GetCurrentConfig()
	index := slices.IndexFunc(config.Albums, func(a player.Album) bool { return a.Title == album.Title })
	last := len(config.Albums) - 1

	//remove album
	config.Albums[index] = config.Albums[last]
	config.Albums = config.Albums[:last]

	//remove album icon
	if err := os.Remove(resource.GetIconPath(album)); err != nil && !os.IsNotExist(err) {
		return err
	}

	return resource.ReloadCurrentConfig()
}

func makeFilter(title string) func(*player.Album) bool {
	title = strings.ToLower(title)
	return func(a *player.Album) bool {
		return strings.Contains(strings.ToLower(a.Title), title)
	}
}

func makeTitleSorter(reverse bool) func(player.Album, player.Album) bool {
	return func(a1, a2 player.Album) bool {
		return (strings.Compare(strings.ToLower(a1.Title), strings.ToLower(a2.Title)) < 0) != reverse
	}
}

func makeDateSorter(reverse bool) func(player.Album, player.Album) bool {
	return func(a1, a2 player.Album) bool {
		return a1.Date.After(a2.Date) != reverse
	}
}
