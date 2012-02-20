// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path"
	"time"
)

import "walk"

type FileInfo struct {
	Name     string
	Size     int64
	Modified int64
}

type FileInfoModel struct {
	items               []*FileInfo
	rowsResetPublisher  walk.EventPublisher
	rowChangedPublisher walk.IntEventPublisher
}

func (m *FileInfoModel) Columns() []walk.TableColumn {
	return []walk.TableColumn{
		{Title: "Name", Width: 200},
		{Title: "Size", Format: "%d", Alignment: walk.AlignFar, Width: 80},
		{Title: "Modified", Format: "2006-01-02 15:04:05", Width: 120},
	}
}

func (m *FileInfoModel) RowCount() int {
	return len(m.items)
}

func (m *FileInfoModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.Name

	case 1:
		return item.Size

	case 2:
		return time.SecondsToLocalTime(item.Modified)
	}

	panic("unexpected col")
}

func (m *FileInfoModel) RowsReset() *walk.Event {
	return m.rowsResetPublisher.Event()
}

func (m *FileInfoModel) RowChanged() *walk.IntEvent {
	return m.rowChangedPublisher.Event()
}

func (m *FileInfoModel) ResetRows(dirPath string) os.Error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	m.items = make([]*FileInfo, 0, len(names))

	for _, name := range names {
		if !excludePath(name) {
			fullPath := path.Join(dirPath, name)

			fi, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			item := &FileInfo{
				Name:     name,
				Size:     fi.Size,
				Modified: fi.Mtime_ns / 1e9,
			}

			m.items = append(m.items, item)
		}
	}

	m.rowsResetPublisher.Publish()

	return nil
}

type MainWindow struct {
	*walk.MainWindow
	fileInfoModel *FileInfoModel
	treeView      *walk.TreeView
	selTvwItem    *walk.TreeViewItem
	tableView     *walk.TableView
	preview       *walk.WebView
}

func (mw *MainWindow) showError(err os.Error) {
	if err == nil {
		return
	}

	walk.MsgBox(mw, "Error", err.String(), walk.MsgBoxOK|walk.MsgBoxIconError)
}

func (mw *MainWindow) populateTreeViewItem(parent *walk.TreeViewItem) {
	mw.treeView.SetSuspended(true)
	defer mw.treeView.SetSuspended(false)

	// Remove dummy child
	parent.Children().Clear()

	dirPath := pathForTreeViewItem(parent)

	dir, err := os.Open(dirPath)
	if err != nil {
		mw.showError(err)
		return
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	panicIfErr(err)

	for _, name := range names {
		fi, err := os.Stat(path.Join(dirPath, name))
		panicIfErr(err)

		if !excludePath(name) && fi.IsDirectory() {
			child := newTreeViewItem(name)

			parent.Children().Add(child)
		}
	}
}

func panicIfErr(err os.Error) {
	if err != nil {
		panic(err)
	}
}

func pathForTreeViewItem(item *walk.TreeViewItem) string {
	var parts []string
	for item != nil {
		parts = append([]string{item.Text()}, parts...)
		item = item.Parent()
	}

	return path.Join(parts...)
}

func excludePath(path string) bool {
	if path == "System Volume Information" {
		return true
	}

	return false
}

func newTreeViewItem(text string) *walk.TreeViewItem {
	item := walk.NewTreeViewItem()
	item.SetText(text)

	// For now, we add a dummy child to make the item expandable.
	item.Children().Add(walk.NewTreeViewItem())

	return item
}

func main() {
	walk.Initialize(walk.InitParams{PanicOnError: true})
	defer walk.Shutdown()

	mainWnd, _ := walk.NewMainWindow()

	mw := &MainWindow{
		MainWindow:    mainWnd,
		fileInfoModel: &FileInfoModel{},
	}

	mw.SetTitle("Walk File Browser Example")
	mw.SetLayout(walk.NewHBoxLayout())

	fileMenu, _ := walk.NewMenu()
	fileMenuAction, _ := mw.Menu().Actions().AddMenu(fileMenu)
	fileMenuAction.SetText("&File")

	exitAction := walk.NewAction()
	exitAction.SetText("E&xit")
	exitAction.Triggered().Attach(func() { walk.App().Exit(0) })
	fileMenu.Actions().Add(exitAction)

	helpMenu, _ := walk.NewMenu()
	helpMenuAction, _ := mw.Menu().Actions().AddMenu(helpMenu)
	helpMenuAction.SetText("&Help")

	aboutAction := walk.NewAction()
	aboutAction.SetText("&About")
	aboutAction.Triggered().Attach(func() {
		walk.MsgBox(mw, "About", "Walk File Browser Example", walk.MsgBoxOK|walk.MsgBoxIconInformation)
	})
	helpMenu.Actions().Add(aboutAction)

	splitter, _ := walk.NewSplitter(mw)

	mw.treeView, _ = walk.NewTreeView(splitter)

	mw.treeView.ItemExpanded().Attach(func(item *walk.TreeViewItem) {
		children := item.Children()
		if children.Len() == 1 && children.At(0).Text() == "" {
			mw.populateTreeViewItem(item)
		}
	})

	mw.treeView.SelectionChanged().Attach(func(old, new *walk.TreeViewItem) {
		mw.selTvwItem = new
		mw.showError(mw.fileInfoModel.ResetRows(pathForTreeViewItem(new)))
	})

	drives, _ := walk.DriveNames()

	mw.treeView.SetSuspended(true)
	for _, drive := range drives {
		driveItem := newTreeViewItem(drive[:2])
		mw.treeView.Items().Add(driveItem)
	}
	mw.treeView.SetSuspended(false)

	mw.tableView, _ = walk.NewTableView(splitter)
	mw.tableView.SetModel(mw.fileInfoModel)
	mw.tableView.SetSingleItemSelection(true)

	mw.tableView.CurrentIndexChanged().Attach(func() {
		var url string

		index := mw.tableView.CurrentIndex()
		if index > -1 {
			name := mw.fileInfoModel.items[index].Name
			url = path.Join(pathForTreeViewItem(mw.selTvwItem), name)
		}

		mw.preview.SetURL(url)
	})

	mw.preview, _ = walk.NewWebView(splitter)

	mw.SetMinMaxSize(walk.Size{600, 400}, walk.Size{})
	mw.SetSize(walk.Size{800, 600})
	mw.Show()

	mw.Run()
}
