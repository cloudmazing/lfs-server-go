package main

/*
Downloader implements a service call to an external entity that will respond with a boolean response
to allow/disallow a user to access a given project's objects
*/
import (
	"io/ioutil"
	"net/http"
)

type Downloader struct {
	Auth     *UserServiceAuth
	Url      string
	Response []byte
	Status   string
}

// interface to hold the response
type Responder interface {
	Status(string) error
	RawResponse([]byte) error
}

func (d *Downloader) GetPage() error {
	// Already set, bounce out. Used for testing
	if d.Response != nil {
		return nil
	}
	resp, err := http.Get(d.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d.Status = resp.Status
	buf, _ := ioutil.ReadAll(resp.Body)
	d.Response = buf
	return nil
}

func NewDownloader(url string) *Downloader {
	return &Downloader{Url: url}
}
