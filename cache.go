package myutil

import (
	"encoding/gob"
	go_cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var Cache *go_cache.Cache

func init() {
	cache_file := "./Cache.gob"
	_, err := os.Lstat(cache_file)
	var M map[string]go_cache.Item
	if !os.IsNotExist(err) {
		File, _ := os.Open(cache_file)
		D := gob.NewDecoder(File)
		D.Decode(&M)
	}
	if len(M) > 0 {
		Cache = go_cache.NewFrom(go_cache.NoExpiration, 10*time.Minute, M)
	} else {
		Cache = go_cache.New(go_cache.NoExpiration, 10*time.Minute)
	}
	go func() {
		for {
			time.Sleep(time.Duration(1) * time.Second)
			File, _ := os.OpenFile(cache_file, os.O_RDWR|os.O_CREATE, 0777)
			defer File.Close()
			enc := gob.NewEncoder(File)
			if err := enc.Encode(Cache.Items()); err != nil {
				log.Error(err)
			}
		}
	}()
}
