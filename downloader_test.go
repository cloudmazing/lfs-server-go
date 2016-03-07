package main

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestDownloaderTestLoads(t *testing.T) {
	assert.Equal(t, true, true)
}

var dlSubject = NewDownloader("http://somewhere.out.there")

func TestDownloaderCreated(t *testing.T) {
	dlSubject.GetPage() // just make sure we dont puke
}
