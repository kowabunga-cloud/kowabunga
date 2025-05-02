/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"github.com/lima-vm/go-qcow2reader/image"
	"github.com/lima-vm/go-qcow2reader/image/qcow2"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

func qcowEncryptionMethod(method qcow2.CryptMethod) string {
	switch method {
	case qcow2.CryptMethodNone:
		return "unencrypted"
	case qcow2.CryptMethodAES:
		return "AES-encrypted"
	case qcow2.CryptMethodLUKS:
		return "LUKS-encrypted"
	}

	return ""
}

func qcowCompressionType(ct qcow2.CompressionType) string {
	switch ct {
	case qcow2.CompressionTypeZlib:
		return "zlib-compressed"
	case qcow2.CompressionTypeZstd:
		return "zstd-compressed"
	}

	return ""
}

func qcowGetImageInformations(fname string, img image.Image) {
	qc := img.(*qcow2.Qcow2)
	version := qc.Version
	cryptMethod := qcowEncryptionMethod(qc.CryptMethod)

	compressionType := qcowCompressionType(qcow2.CompressionTypeZlib) // default if unspecified
	if qc.HeaderFieldsAdditional != nil {
		compressionType = qcowCompressionType(qc.CompressionType)
	}
	klog.Infof("%s is a QCOW2 v%d disk image (%s, %s)", fname, version, cryptMethod, compressionType)

	// check for possible Zstd compression
	if qc.HeaderFieldsAdditional != nil && qc.CompressionType == qcow2.CompressionTypeZstd {
		klog.Debugf("QCOW2: registering ZSTD stream decompressor")
		qcow2.SetDecompressor(qcow2.CompressionTypeZstd, NewZstdDecompressor)
	}
}
