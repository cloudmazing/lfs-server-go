package main

import (
	"github.com/memikequinn/lfs-server-go/Godeps/_workspace/src/github.com/bmizerany/assert"
	"testing"
)

func TestUserServiceTestLoads(t *testing.T) {
	assert.Equal(t, true, true)
}

func TestThatConsumerUserServiceRespondsWhenTrue(t *testing.T) {
	us := setupUs(true)
	assert.Equal(t, true, us.Can())
}

func TestThatTheStructIsFilled(t *testing.T) {
	us := setupUs(true)
	us.GetResponse()
	assert.Equal(t, true, us.Action != "")
	assert.Equal(t, true, us.Username != "")
	assert.Equal(t, true, us.Project != "")
}

func TestThatConsumerUserServiceRespondsWhenFalse(t *testing.T) {
	us := setupUs(false)
	assert.Equal(t, false, us.Can())
}

func TestThatConsumerUserServiceSetsMessageWhenInvalidAction(t *testing.T) {
	us := NewUserService("http://somewhere.net", "testuser", "testproject", "poo")
	assert.Equal(t, "poo is not in AllowedActions", us.UserAccessResponse.Message)
}

func TestConsumerUserServiceResponds_Can(t *testing.T) {
	us := NewUserService("http://somewhere.net", "testuser", "testproject", "download")
	us.Downloader.Response = mock_get_page("http://somewhere.net", true)
	assert.Equal(t, true, us.Can())
}

func TestDownloaderAccessWhenTrue(t *testing.T) {
	us := setupUs(true)
	assert.Equal(t, true, us.UserAccessResponse.Access)
	assert.Equal(t, "yay", us.UserAccessResponse.Status)
	assert.Equal(t, "Some Message", us.UserAccessResponse.Message)
}

func TestDownloaderAccessWhenFalse(t *testing.T) {
	us := setupUs(false)
	assert.Equal(t, false, us.UserAccessResponse.Access)
	assert.Equal(t, false, us.Can())
	assert.Equal(t, "yay", us.UserAccessResponse.Status)
	assert.Equal(t, "Some Message", us.UserAccessResponse.Message)
}

func setupUs(access bool) *UserService {
	d := &Downloader{Response: mock_get_page("http://somewhere.net", access)}
	uar := &UserAccessResponse{RawResponse: d.Response}
	us := &UserService{Downloader: d, UserAccessResponse: uar}
	us.Action = "download"
	us.Project = "testproject"
	us.Username = "testuser"
	us.GetResponse()
	return us
}
