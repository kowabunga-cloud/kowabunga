/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	DownloaderProgressMsg  = "Downloading %s (%.02f%% completed) - %s"
	DownloaderChecksumMsg  = "Setting SHA256 checksum validation for %s"
	DownloaderCompletedMsg = "File from has been retrieved from %s at %.02f Mbps"
)

func DownloadFromURL(url, dst, csum string) error {
	klog.Infof("Downloading resource from %s ...", url)

	client := grab.NewClient()
	req, err := grab.NewRequest("", url)
	if err != nil {
		return err
	}
	req.Filename = dst

	if csum != "" {
		sum, err := hex.DecodeString(csum)
		if err != nil {
			return err
		}
		klog.Infof(DownloaderChecksumMsg, url)
		req.SetChecksum(sha256.New(), sum, true)
	}

	resp := client.Do(req)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			eta := time.Until(resp.ETA()).Seconds()
			var eta_msg string
			if eta == 0 {
				eta_msg = "DONE"
			} else if eta < 0 {
				eta_msg = "ETA being calculated"
			} else {
				eta_msg = fmt.Sprintf("ETA in %d second(s)", int(eta))
			}
			klog.Infof(DownloaderProgressMsg, url, resp.Progress()*100, eta_msg)

		case <-resp.Done:
			if resp.Err() != nil {
				return err
			}
			klog.Infof(DownloaderProgressMsg, url, resp.Progress()*100, "DONE")
			bps := resp.BytesPerSecond() / MiB * 8
			klog.Infof(DownloaderCompletedMsg, url, bps)

			return resp.Err()
		}
	}
}
