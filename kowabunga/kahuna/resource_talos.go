/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	b64 "encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
)

const (
	MongoCollectionTalosSchemaVersion = 1
	MongoCollectionTalosName          = "talos"
	// Talos name will always result in talos-<regionname>
	TalosDefaultNamePrefix = "talos"
	TalosProfileName       = "talos"

	TalosCtlBinaryUrl   = "https://github.com/siderolabs/talos/releases/download/%s/talosctl-%s-%s"
	TalosCtlChecksumUrl = "https://github.com/siderolabs/talos/releases/download/%s/sha256sum.txt"
	TalosCtlBinaryName  = "/usr/bin/talosctl-%s"
	TalosInstallDisk    = "/dev/vda"
	TalosWorkingDir     = "/tmp/%s"
	TalosSecretFilename = "secrets.yaml"
)

type Talos struct {
	// anonymous field, inheritance
	Resource `bson:"inline"`

	// parents
	ProjectID string `bson:"project_id"`

	// children references
	MultiZonesResourceID string `bson:"mzr_id"`

	// secret content
	Secrets string `bson:"secrets"`
}

func NewTalos(projectId, regionId, desc, name, talosVersion string, cpu, memory, disk int64) (*Talos, error) {
	// TODO: on input validate version through regex
	// Download binary first to fail fast if it does not exist
	err := downloadTalosctlBinary(talosVersion)
	if err != nil {
		return nil, fmt.Errorf("cannot download the specified Talos Version. Check if the release exists. %s", err.Error())
	}

	mzr, err := NewMultiZonesResource(projectId, regionId, TalosDefaultNamePrefix,
		desc, TalosProfileName, TalosProfileName, cpu, memory, disk, 0, "")
	if err != nil {
		mzr.Delete()
		return nil, err
	}

	workingDir := fmt.Sprintf(TalosWorkingDir, name)
	os.Mkdir(workingDir, 700)
	secretFilePath, err := generateTalosSecrets(workingDir, talosVersion, name)
	if err != nil {
		_ = os.RemoveAll(workingDir)
		_ = mzr.Delete()
		return nil, err
	}
	secretsContent, err := os.ReadFile(secretFilePath)
	if err != nil {
		_ = os.RemoveAll(workingDir)
		_ = mzr.Delete()
		return nil, err
	}
	b64Secrets := b64.StdEncoding.EncodeToString([]byte(secretsContent))
	talosConfigFile, controlplaneConfigFile, err := generateTalosControlplaneConfig(workingDir, talosVersion, secretFilePath, name, mzr.PrivateVIPs[0])

	t := &Talos{MultiZonesResourceID: mzr.String(), Secrets: b64Secrets}

	t.initCluster()
	os.RemoveAll(workingDir)
	return t, nil
}

func (*Talos) initCluster() {
	//TODO: bootstrap etcd on 1 node. And that's all folks. Optionnaly deploy cilium ?
}

func downloadTalosctlBinary(version string) error {
	checksumSrc := fmt.Sprintf(TalosCtlChecksumUrl, version)
	dst := fmt.Sprintf(TalosCtlBinaryName, version)
	url := fmt.Sprintf(TalosCtlBinaryUrl, version, runtime.GOOS, runtime.GOARCH)

	// Pulling the checksum list to check the binary signature
	res, err := http.Get(checksumSrc)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()

	var csum string
	versionedTalosctlBinaryName := fmt.Sprintf("talosctl-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, line := range strings.Split(string(body), "\n") {
		versionChecksumPair := strings.Fields(line)
		if len(versionChecksumPair) > 1 {
			if versionChecksumPair[1] == versionedTalosctlBinaryName {
				csum = versionChecksumPair[0]
				break
			}
		}
	}
	err = common.DownloadFromURL(url, dst, csum)
	if err != nil {
		// Ignore errors here.
		_ = os.Remove(dst)
		return err
	}
	return nil
}

func generateTalosSecrets(workingDir, talosVersion, clusterName string) (string, error) {
	versionedTalosCtlBinary := fmt.Sprintf(TalosCtlBinaryName, talosVersion)
	targetFile := fmt.Sprintf("%s/%s", workingDir, TalosSecretFilename)
	err := common.BinExec(versionedTalosCtlBinary, "",
		[]string{"gen", "secrets",
			"--talos-version", talosVersion,
			"--output-file", targetFile},
		nil)
	if err != nil {
		return "", err
	}
	return targetFile, nil
}
func generateTalosControlplaneConfig(workingDir, talosVersion, secretFilePath, clusterName, vip string) (talosConfigFile, controlplaneConfigFile string, err error) {
	versionedTalosCtlBinary := fmt.Sprintf(TalosCtlBinaryName, talosVersion)
	patchDisableDefaultCNI := fmt.Sprintf("[{'op': 'add', 'path': '/machine/network/interfaces/0/vip/ip', 'value': '%s'},{'op': 'add', 'path': '/cluster/proxy/disabled', 'value': true},{'op': 'add', 'path': '/cluster/network', 'value': {'cni': {'name': 'none'}}}]", vip)
	err = common.BinExec(versionedTalosCtlBinary, "",
		[]string{"gen", "config", clusterName, fmt.Sprintf("https://%s:6443", vip),
			"--output-types", "talosconfig,controlplane",
			"--talos-version", talosVersion,
			"--install-disk", TalosInstallDisk,
			"--config-patch-control-plane", patchDisableDefaultCNI,
			"--with-secrets", secretFilePath,
			"--output-dir", workingDir},
		nil)
	if err != nil {
		return "", "", err
	}
	talosConfigFile = fmt.Sprintf("/tmp/%s/controlplane.yaml", clusterName)
	controlplaneConfigFile = fmt.Sprintf("/tmp/%s/controlplane.yaml", clusterName)
	return talosConfigFile, controlplaneConfigFile, nil
}
