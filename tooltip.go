// Copyright 2010 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package walk

import (
	"syscall"
	"unsafe"
)

import . "github.com/lxn/go-winapi"

type ToolTip struct {
	WidgetBase
}

func NewToolTip(parent Container) (*ToolTip, error) {
	tt := &ToolTip{}

	if err := initWidget(
		tt,
		parent,
		"tooltips_class32",
		WS_POPUP|TTS_ALWAYSTIP|TTS_BALLOON,
		WS_EX_TOPMOST); err != nil {
		return nil, err
	}

	succeeded := false
	defer func() {
		if !succeeded {
			tt.Dispose()
		}
	}()

	if err := parent.Children().Add(tt); err != nil {
		return nil, err
	}

	SetWindowPos(tt.hWnd, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)

	succeeded = true

	return tt, nil
}

func (*ToolTip) LayoutFlags() LayoutFlags {
	return 0
}

func (tt *ToolTip) SizeHint() Size {
	return Size{0, 0}
}

func (tt *ToolTip) Title() string {
	var gt TTGETTITLE

	buf := make([]uint16, 128)

	gt.DwSize = uint32(unsafe.Sizeof(gt))
	gt.Cch = uint32(len(buf))
	gt.PszTitle = &buf[0]

	SendMessage(tt.hWnd, TTM_GETTITLE, 0, uintptr(unsafe.Pointer(&gt)))

	return syscall.UTF16ToString(buf)
}

func (tt *ToolTip) SetTitle(value string) error {
	if FALSE == SendMessage(tt.hWnd, TTM_SETTITLE, uintptr(TTI_INFO), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(value)))) {
		return newError("TTM_SETTITLE failed")
	}

	return nil
}

func (tt *ToolTip) AddWidget(widget Widget, text string) error {
	var ti TOOLINFO

	ti.CbSize = uint32(unsafe.Sizeof(ti))
	parent := widget.Parent()
	if parent != nil {
		ti.Hwnd = parent.BaseWidget().hWnd
	}
	ti.UFlags = TTF_IDISHWND | TTF_SUBCLASS
	ti.UId = uintptr(widget.BaseWidget().hWnd)
	ti.LpszText = syscall.StringToUTF16Ptr(text)

	if FALSE == SendMessage(tt.hWnd, TTM_ADDTOOL, 0, uintptr(unsafe.Pointer(&ti))) {
		return newError("TTM_ADDTOOL failed")
	}

	return nil
}

func (tt *ToolTip) RemoveWidget(widget Widget) error {
	panic("not implemented")
}
