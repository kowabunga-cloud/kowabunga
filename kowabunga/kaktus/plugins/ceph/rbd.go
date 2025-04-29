/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"io"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/kaktus"
)

const (
	CephRwBufferSize               = 4 * common.MiB
	CephOsImageSnapshotName        = "kowabunga"
	CephEnableExperimentalFeatures = false
)

func (ceph *ccs) ListRbdVolumes(poolName string) ([]string, error) {
	iocon, err := ceph.Conn.OpenIOContext(poolName)
	if err != nil {
		return []string{}, err
	}

	return rbd.GetImageNames(iocon)
}

func (ceph *ccs) GetPoolStats(poolName string) (uint64, uint64, uint64, error) {
	iocon, err := ceph.Conn.OpenIOContext(poolName)
	if err != nil {
		return 0, 0, 0, err
	}

	poolStats, err := iocon.GetPoolStats()
	if err != nil {
		return 0, 0, 0, err
	}

	clusterStats, err := ceph.Conn.GetClusterStats()
	if err != nil {
		return 0, 0, 0, err
	}

	return poolStats.Num_bytes, clusterStats.Kb_avail * common.KiB, clusterStats.Kb * common.KiB, nil
}

// don't forget to close image once done
func (ceph *ccs) getImage(poolName, volName string) (*rados.IOContext, *rbd.Image, error) {
	iocon, err := ceph.Conn.OpenIOContext(poolName)
	if err != nil {
		return nil, nil, err
	}

	img, err := rbd.OpenImage(iocon, volName, "")
	if err != nil {
		klog.Errorf("unable to open RBD volume %s from pool %s: %v", volName, poolName, err)
		return nil, nil, err
	}

	return iocon, img, nil
}

func (ceph *ccs) GetRbdVolumeInfos(poolName, volName string) (uint64, error) {
	_, img, err := ceph.getImage(poolName, volName)
	if err != nil {
		return 0, err
	}
	defer img.Close()

	size, err := img.GetSize()
	if err != nil {
		klog.Errorf("unable to get RBD volume %s size: %v", volName, err)
		return 0, err
	}

	return size, nil
}

func (ceph *ccs) newRbdVolume(poolName, volName string, size uint64) (*rbd.Image, error) {
	iocon, err := ceph.Conn.OpenIOContext(poolName)
	if err != nil {
		return nil, err
	}

	klog.Infof("Creating RBD volume %s of %d bytes ...", volName, size)

	rio := rbd.NewRbdImageOptions()
	err = rio.SetUint64(rbd.ImageOptionFormat, 2)
	if err != nil {
		klog.Errorf("unable to create RBD volume %s on pool %s: %v", volName, poolName, err)
		return nil, err
	}

	err = rbd.CreateImage(iocon, volName, size, rio)
	if err != nil {
		klog.Errorf("unable to create RBD volume %s on pool %s: %v", volName, poolName, err)
		return nil, err
	}

	img, err := rbd.OpenImage(iocon, volName, "")
	if err != nil {
		klog.Errorf("unable to open RBD volume %s from pool %s: %v", volName, poolName, err)
		return nil, err
	}

	if CephEnableExperimentalFeatures {
		// set default features
		features := []string{
			rbd.FeatureNameLayering,
			rbd.FeatureNameExclusiveLock,
			rbd.FeatureNameObjectMap,
			rbd.FeatureNameFastDiff,
			rbd.FeatureNameDeepFlatten,
			rbd.FeatureNameOperations,
		}
		featuresSet := rbd.FeatureSetFromNames(features)
		err = img.UpdateFeatures(uint64(featuresSet), true)
		if err != nil {
			klog.Errorf("unable to set RBD volume %s features: %v", volName, err)
		}
	}

	klog.Debugf("RBD volume %s successfully created", volName)
	return img, nil
}

func (ceph *ccs) CreateRbdVolume(poolName, volName string, size uint64) error {
	klog.Debugf("Opening volume %s ...", volName)
	img, err := ceph.newRbdVolume(poolName, volName, size)
	if err != nil {
		return err
	}

	err = img.Close()
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func (ceph *ccs) writeImageData(img *rbd.Image, size uint64, data []byte) error {
	var dsize uint64 = uint64(len(data))
	if dsize != size {
		err := fmt.Errorf("mismatch between RBD volume size (%d) and data size (%d)", size, dsize)
		klog.Error(err)
		return err
	}

	written, err := img.Write(data)
	if err != nil {
		return err
	}

	if uint64(written) != size {
		err := fmt.Errorf("mismatch between RBD volume size (%d) and written data (%d)", size, written)
		klog.Error(err)
		return err
	}

	err = img.Flush()
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func (ceph *ccs) CreateRbdVolumeFromBinData(poolName, volName string, size uint64, data []byte) error {
	img, err := ceph.newRbdVolume(poolName, volName, size)
	if err != nil {
		return err
	}
	defer img.Close()

	err = ceph.writeImageData(img, size, data)
	if err != nil {
		return err
	}

	err = img.Flush()
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func (ceph *ccs) UpdateRbdVolumeFromBinData(poolName, volName string, size uint64, data []byte) error {
	err := ceph.ResizeRbdVolume(poolName, volName, size)
	if err != nil {
		return err
	}

	_, img, err := ceph.getImage(poolName, volName)
	if err != nil {
		return err
	}
	defer img.Close()

	err = ceph.writeImageData(img, size, data)
	if err != nil {
		return err
	}

	err = img.Flush()
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func (ceph *ccs) CreateRbdVolumeFromUrl(poolName, volName, url string) (uint64, error) {
	tmpImage, err := common.NewTmpFile("")
	if err != nil {
		return 0, err
	}
	defer tmpImage.Remove()

	diskImage, err := kaktus.NewDiskImageFromURL(url, tmpImage.File(), "")
	if err != nil {
		return 0, err
	}

	tmpRawImage, err := common.NewTmpFile("")
	if err != nil {
		return 0, err
	}
	defer tmpRawImage.Remove()

	err = diskImage.ToRaw(tmpRawImage.File(), true)
	if err != nil {
		return 0, err
	}

	img, err := ceph.newRbdVolume(poolName, volName, diskImage.Size())
	if err != nil {
		return 0, err
	}
	defer img.Close()

	buf := make([]byte, CephRwBufferSize)
	var offset int64
	for {
		n, err := tmpRawImage.File().ReadAt(buf, offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		if n > 0 {
			_, err := img.WriteAt(buf[:n], offset)
			if err != nil {
				klog.Errorf("unable to write content volume to %s on pool %s: %v", volName, poolName, err)
				return 0, err
			}
			offset += int64(n)
			fmt.Printf("\rWriting volume %s on pool %s: %.2f%% completed", volName, poolName, float64(offset)/float64(diskImage.Size())*100)
		}
	}

	err = img.Flush()
	if err != nil {
		klog.Error(err)
		return 0, err
	}

	return diskImage.Size(), nil
}

func (ceph *ccs) CloneRbdVolume(poolName, srcName, dstName string, size uint64) error {
	iocon, img, err := ceph.getImage(poolName, srcName)
	if err != nil {
		return err
	}
	defer img.Close()

	// ensure source image snapshot has a valid snapshot, create it otherwise
	snaps, err := img.GetSnapshotNames()
	if err != nil {
		klog.Errorf("unable to list snapshots from RBD volume %s: %v", srcName, err)
		return err
	}

	requiresSnapshotCreation := true
	for _, s := range snaps {
		if s.Name == CephOsImageSnapshotName {
			requiresSnapshotCreation = false
			break
		}
	}

	if requiresSnapshotCreation {
		snap, err := img.CreateSnapshot(CephOsImageSnapshotName)
		if err != nil {
			klog.Errorf("unable to create snapshot from RBD volume %s: %v", srcName, err)
			return err
		}

		err = snap.Protect()
		if err != nil {
			klog.Errorf("unable to protect snapshot from RBD volume %s: %v", srcName, err)
			return err
		}
	}

	rio := rbd.NewRbdImageOptions()
	err = rio.SetUint64(rbd.ImageOptionFormat, 2)
	if err != nil {
		return err
	}

	klog.Debugf("Cloning volume %s into %s ...", srcName, dstName)
	err = rbd.CloneImage(iocon, srcName, CephOsImageSnapshotName, iocon, dstName, rio)
	if err != nil {
		klog.Errorf("unable to clone RBD volume %s from pool %s into volume %s: %v", srcName, poolName, dstName, err)
		return err
	}

	return ceph.ResizeRbdVolume(poolName, dstName, size)
}

func (ceph *ccs) ResizeRbdVolume(poolName, volName string, size uint64) error {
	_, img, err := ceph.getImage(poolName, volName)
	if err != nil {
		return err
	}
	defer img.Close()

	klog.Infof("Resizing RBD volume %s from pool %s to %d bytes ...", volName, poolName, size)
	err = img.Resize(size)
	if err != nil {
		klog.Errorf("unable to resize RBD volume %s from pool %s: %v", volName, poolName, err)
		return err
	}

	return nil
}

func (ceph *ccs) DeleteRbdVolume(poolName, volName string, deleteSnapshots bool) error {
	_, img, err := ceph.getImage(poolName, volName)
	if err != nil {
		return err
	}
	defer img.Close()

	if deleteSnapshots {
		// list all snapshots, if any
		snaps, err := img.GetSnapshotNames()
		if err != nil {
			klog.Errorf("unable to get volume %s snapshot names: %v", volName, err)
			return err
		}

		// remove snapshots, if any
		for _, s := range snaps {
			snapshot := img.GetSnapshot(s.Name)
			if snapshot == nil {
				continue
			}

			protected, err := snapshot.IsProtected()
			if err != nil {
				klog.Errorf("unable to get volume %s snapshot %s protection state: %v", volName, s.Name, err)
				return err
			}

			if protected {
				err := snapshot.Unprotect()
				if err != nil {
					klog.Errorf("unable to unprotect volume %s snapshot %s: %v", volName, s.Name, err)
					return err
				}
			}

			klog.Infof("Removing RBD volume %s snapshot %s ...", volName, s.Name)
			err = snapshot.Remove()
			if err != nil {
				klog.Errorf("unable to delete volume %s snapshot %s: %v", volName, s.Name, err)
				return err
			}
		}
	}

	// enforce release of volume locks, if any
	lockOwners, err := img.LockGetOwners()
	if err != nil {
		klog.Errorf("unable to get lock owners on volume %s: %v", volName, err)
		return err
	}

	for _, owner := range lockOwners {
		err := img.LockBreak(owner.Mode, owner.Owner)
		if err != nil {
			klog.Errorf("unable to release lock on volume %s for %s: %v", volName, owner.Owner, err)
			return err
		}
	}

	// close image
	err = img.Close()
	if err != nil {
		klog.Error(err)
		return err
	}

	klog.Infof("Removing RBD volume %s from pool %s ...", volName, poolName)
	err = img.Remove()
	if err != nil {
		klog.Errorf("unable to remove RBD volume %s from pool %s: %v", volName, poolName, err)
		return err
	}

	return nil
}
