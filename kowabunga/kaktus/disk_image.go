/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/lima-vm/go-qcow2reader"
	"github.com/lima-vm/go-qcow2reader/image"
	"github.com/lima-vm/go-qcow2reader/image/qcow2"
	"github.com/lima-vm/go-qcow2reader/image/raw"
	"github.com/machinebox/progress"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	DiskImageUnsupportedTypeError = "Unsupported disk image type: %s"
	DiksImageRawWriteError        = "Mismatch between virtual image disk size and raw written bytes"
)

var supportedDiskImageTypes = []image.Type{
	qcow2.Type,
	raw.Type,
}

type DiskImage struct {
	name string
	img  image.Image
}

func (img *DiskImage) Size() uint64 {
	return uint64(img.img.Size())
}

func (img *DiskImage) Detect() {
	switch img.img.Type() {
	case qcow2.Type:
		qcowGetImageInformations(img.name, img.img)
	}
}

func (img *DiskImage) ToRaw(dst *os.File, displayProgression bool) error {
	klog.Infof("Converting %s-formatted image %s to RAW-format %s", img.img.Type(), img.name, dst.Name())
	imgReader := io.NewSectionReader(img.img, 0, img.img.Size())

	var writtenBytes int64
	if displayProgression {
		progressReader := progress.NewReader(imgReader)
		go func() {
			progressChan := progress.NewTicker(context.Background(), progressReader, img.img.Size(), 1*time.Second)
			for p := range progressChan {
				klog.Debugf("%s raw-conversion (%.02f%% completed)", img.name, p.Percent())
			}
		}()

		n, err := io.Copy(dst, progressReader)
		if err != nil {
			return err
		}
		writtenBytes = n
	} else {
		n, err := io.Copy(dst, imgReader)
		if err != nil {
			return err
		}
		writtenBytes = n
	}

	if writtenBytes != img.img.Size() {
		return fmt.Errorf("%s", DiksImageRawWriteError)
	}

	return nil
}

func newDiskImage(file *os.File) (*DiskImage, error) {
	img, err := qcow2reader.Open(file)
	if err != nil {
		return nil, err
	}

	di := DiskImage{
		name: file.Name(),
		img:  img,
	}

	if !slices.Contains(supportedDiskImageTypes, img.Type()) {
		return &di, fmt.Errorf(DiskImageUnsupportedTypeError, img.Type())
	}

	// auto-detect
	di.Detect()

	return &di, nil
}

func NewDiskImageFromFile(file *os.File) (*DiskImage, error) {
	return newDiskImage(file)
}

func NewDiskImageFromURL(url string, dst *os.File, checkSum string) (*DiskImage, error) {
	klog.Infof("Downloading %s into %s ...", url, dst.Name())
	err := common.DownloadFromURL(url, dst.Name(), checkSum)
	if err != nil {
		return nil, err
	}

	return newDiskImage(dst)
}
