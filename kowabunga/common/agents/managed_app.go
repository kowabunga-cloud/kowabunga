/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
	"text/template"

	dbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
)

const (
	SystemdActiveState             = "active"
	ErrorSystemdConnectionUserConn = "Has the user appropriate rights to interact with systemd ?"
)

type ManagedService struct {
	BinaryPath  string // Not used for now. Later used for upgrade/update
	UnitName    string
	User        string
	Group       string
	ConfigPaths []ConfigFile
	Pre         []func(metadata *metadata.InstanceMetadata, args ...any) error
	Post        []func(metadata *metadata.InstanceMetadata, args ...any) error
	Reload      []func(metadata *metadata.InstanceMetadata, args ...any) error
}

type ConfigFile struct {
	TemplateContent string
	TargetPath      string
	IsExecutable    bool
}

func (svc *ManagedService) ReloadOrRestart(metadata *metadata.InstanceMetadata) error {
	ctx := context.Background()
	systemdConnection, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return fmt.Errorf("%s -- %s", err, ErrorSystemdConnectionUserConn)
	}
	defer systemdConnection.Close()
	code, err := systemdConnection.ReloadOrRestartUnitContext(ctx, svc.UnitName, "replace", nil)
	if err != nil {
		return fmt.Errorf("%s | returned code : %d", err.Error(), code)
	}
	klog.Infof("Systemd unit %s has been reloaded (or restarted)", svc.UnitName)
	for _, reloadFunc := range svc.Reload {
		err := reloadFunc(metadata)
		if err != nil {
			return err
		}
	}
	return nil
}

// values: a map of values to inject for each template for each ConfigPath
func (svc *ManagedService) TemplateConfigs(values map[string]any) (bool, error) {
	configUpdated := false
	for _, cfg := range svc.ConfigPaths {
		tpl := template.New(cfg.TargetPath)
		common.LoadTemplateFunctions(tpl)
		tpl, err := tpl.Parse(cfg.TemplateContent)
		if err != nil {
			return configUpdated, err
		}

		var buffer bytes.Buffer
		err = tpl.Execute(&buffer, values)
		if err != nil {
			klog.Error(err)
		}

		content := buffer.Bytes()
		diff, err := hasDiff(content, cfg.TargetPath)
		if err != nil {
			return configUpdated, err
		}
		if diff {
			configUpdated = true

			us, err := user.Lookup(svc.User)
			if err != nil {
				return configUpdated, err
			}
			uid, err := strconv.Atoi(us.Uid)
			if err != nil {
				return configUpdated, err
			}

			grp, err := user.LookupGroup(svc.Group)
			if err != nil {
				return configUpdated, err
			}
			gid, err := strconv.Atoi(grp.Gid)
			if err != nil {
				return configUpdated, err
			}

			targetDir := filepath.Dir(cfg.TargetPath)

			err = os.MkdirAll(targetDir, 0750)
			if err != nil {
				return configUpdated, err
			}

			err = os.Chown(targetDir, uid, gid)
			if err != nil {
				return configUpdated, err
			}

			var filemode os.FileMode = 0600
			if cfg.IsExecutable {
				filemode = 0700
			}

			err = os.WriteFile(cfg.TargetPath, buffer.Bytes(), filemode)
			if err != nil {
				return configUpdated, err
			}

			err = os.Chown(cfg.TargetPath, uid, gid)
			if err != nil {
				return configUpdated, err
			}

			klog.Infof("A change has been applied to %s", cfg.TargetPath)
		}
	}
	return configUpdated, nil
}

func (svc *ManagedService) IsServiceStarted() (bool, error) {
	ctx := context.Background()
	systemdConnection, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return false, fmt.Errorf("%s -- %s", err.Error(), ErrorSystemdConnectionUserConn)
	}
	defer systemdConnection.Close()
	state, err := systemdConnection.ListUnitsByNamesContext(ctx, []string{svc.UnitName})
	if err != nil {
		return false, err
	}
	return state[0].ActiveState == SystemdActiveState, nil
}

func hasDiff(newValue []byte, targetOverrideFilePath string) (bool, error) {
	if _, err := os.Stat(targetOverrideFilePath); errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	existingBytes, err := os.ReadFile(filepath.Clean(targetOverrideFilePath))
	if err != nil {
		return false, err
	}
	return !bytes.Equal(newValue, existingBytes), nil
}

func AgentTestTemplate(t *testing.T, services map[string]*ManagedService, cfgDir string, config map[string]any) {
	us, err := user.Current()
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	grp, err := user.LookupGroupId(us.Gid)
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	for _, svc := range services {
		svc.User = us.Username
		svc.Group = grp.Name
		for id, p := range svc.ConfigPaths {
			svc.ConfigPaths[id].TargetPath = cfgDir + "/" + p.TargetPath
		}
		_, err := svc.TemplateConfigs(config)
		if err != nil {
			t.Errorf("%s", err.Error())
		}
	}
}
