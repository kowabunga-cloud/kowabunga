/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"os"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB

	MimeJSON = "application/json"
)

var DiffieHellmanIanaNames = map[int]string{
	2:  "modp1024",
	14: "modp2048",
	15: "modp3072",
	16: "modp4096",
	17: "modp6144",
	18: "modp8192",
	19: "ecp256",
	20: "ecp384",
	21: "ecp521",
	22: "modp1024s160",
	23: "modp2048s224",
	24: "modp2048s256",
}

type TmpFile struct {
	file *os.File
}

func (f *TmpFile) File() *os.File {
	return f.file
}

func (f *TmpFile) Remove() {
	err := f.file.Close()
	if err != nil {
		klog.Error(err)
	}

	klog.Infof("Cleaning leftover %s ...", f.file.Name())
	err = os.Remove(f.file.Name())
	if err != nil {
		klog.Error(err)
	}
}

func NewTmpFile(prefix string) (*TmpFile, error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return nil, err
	}

	return &TmpFile{
		file: f,
	}, nil
}

func IsRoot() bool {
	return os.Getuid() == 0
}
