/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	store "github.com/eko/gocache/lib/v4/store"
	freecache_store "github.com/eko/gocache/store/freecache/v4"

	"github.com/kowabunga-cloud/common"
	"github.com/kowabunga-cloud/common/klog"
)

const (
	CacheTypeInMemory = "memory"

	CacheErrDisabled = "in-memory cache is disabled"

	CacheNsUserResources = "userResources"
)

type KowabungaCache struct {
	enabled bool
	ms      *marshaler.Marshaler
}

// cache singleton
var cacheLock = &sync.Mutex{}
var kCache *KowabungaCache

func GetCache() *KowabungaCache {
	if kCache == nil {
		cacheLock.Lock()
		defer cacheLock.Unlock()
		klog.Debugf("Creating Kowabunga Cache instance")
		kCache = &KowabungaCache{}
	}

	return kCache
}

func (kc *KowabungaCache) Init(enabled bool, tp string, size, ttl int) {
	kc.enabled = enabled

	if !enabled {
		return
	}

	switch tp {
	case CacheTypeInMemory:
		klog.Debugf("Initializing Kowabunga in-memory cache ...")
		sizeMB := size * common.MiB
		expire := time.Duration(ttl) * time.Minute
		fcs := freecache_store.NewFreecache(freecache.NewCache(sizeMB), store.WithExpiration(expire))
		kc.ms = marshaler.New(cache.New[any](fcs))
	default:
		// unsupported cache type, disabling
		kc.enabled = false
	}
}

func (kc *KowabungaCache) key(ns, key string) string {
	return fmt.Sprintf("%s/%s", ns, key)
}

func (kc *KowabungaCache) Set(ns, key string, value any) {
	if !kc.enabled {
		return
	}

	err := kc.ms.Set(context.TODO(), kc.key(ns, key), value)
	if err != nil {
		klog.Errorf("Unable to set %s/%s cache value: %v", ns, key, err)
	}
}

func (kc *KowabungaCache) Get(ns, key string, result interface{}) error {
	if !kc.enabled {
		return fmt.Errorf("%s", CacheErrDisabled)
	}

	_, err := kc.ms.Get(context.TODO(), kc.key(ns, key), result)
	return err
}

func (kc *KowabungaCache) Delete(ns, key string) error {
	if !kc.enabled {
		return fmt.Errorf("%s", CacheErrDisabled)
	}

	return kc.ms.Delete(context.TODO(), kc.key(ns, key))
}
