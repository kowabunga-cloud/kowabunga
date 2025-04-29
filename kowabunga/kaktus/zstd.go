/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kaktus

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

type zstdDecompressor struct {
	*zstd.Decoder
}

func (z *zstdDecompressor) Close() error {
	z.Decoder.Close()
	return nil
}

func NewZstdDecompressor(r io.Reader) (io.ReadCloser, error) {
	dec, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &zstdDecompressor{dec}, nil
}
