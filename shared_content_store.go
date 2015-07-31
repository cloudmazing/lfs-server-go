package main

import "path/filepath"

func transformKey(key string) string {
	if len(key) < 5 {
		return key
	}

	return filepath.Join(key[0:2], key[2:4], key[4:len(key)])
}
