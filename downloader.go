package main
import (
	"net/http"
	"io/ioutil"
)

type UserServiceAuth struct {
	Username string
	Password string
}

type Downloader struct {
	Auth     *UserServiceAuth
	Url      string
	Response []byte
	Status string
}

// interface to hold the response
type Responder interface {
	Status(string) error
	RawResponse([]byte) error
}

func (d *Downloader) GetPage() (error) {
	// Already set, bounce out. Used for testing
	if d.Response != nil {return nil}
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
