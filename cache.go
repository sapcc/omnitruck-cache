package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/majewsky/schwift"
	"github.com/majewsky/schwift/gopherschwift"
)

type Cache interface {
	Get(key string) (*url.URL, bool, error)
	Store(key string, file io.ReadSeeker) (*url.URL, error)
}

type LocalCache struct {
	BaseURL  string
	CacheDir string
}

func (l *LocalCache) Get(key string) (*url.URL, bool, error) {

	path := filepath.Join(l.CacheDir, key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, false, nil
	}
	url, err := url.Parse(l.BaseURL + key)
	return url, true, err
}

func (l *LocalCache) Store(key string, tmpfile io.ReadSeeker) (*url.URL, error) {
	path := filepath.Join(l.CacheDir, key)

	url, err := url.Parse(l.BaseURL + key)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(file, tmpfile)
	return url, err

}

type SwiftCache struct {
	container *schwift.Container
}

func NewSwiftCache(container string) (*SwiftCache, error) {

	if container == "" {
		return nil, errors.New("Container name required")
	}
	authOptions, err := openstack.AuthOptionsFromEnv()

	if os.Getenv("OS_PROJECT_DOMAIN_NAME") != "" {
		authOptions.Scope = &gophercloud.AuthScope{
			ProjectName: os.Getenv("OS_PROJECT_NAME"),
			DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
		}
	}

	if err != nil {
		return nil, err
	}
	authOptions.AllowReauth = true
	provider, err := openstack.AuthenticatedClient(authOptions)
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	account, err := gopherschwift.Wrap(client, nil)
	if err != nil {
		return nil, err
	}
	c, err := account.Container(container).EnsureExists()
	if err != nil {
		return nil, err
	}
	return &SwiftCache{container: c}, nil
}

func (s *SwiftCache) Get(key string) (*url.URL, bool, error) {
	key = strings.TrimLeft(key, "/")
	object := s.container.Object(key)
	exists, err := object.Exists()
	if err != nil {
		return nil, false, err
	}
	if exists {
		objectURL, err := object.URL()
		if err != nil {
			return nil, false, err
		}
		u, err := url.Parse(objectURL)
		return u, true, err
	}
	return nil, false, nil
}

func (s *SwiftCache) Store(key string, tmpfile io.ReadSeeker) (*url.URL, error) {

	key = strings.TrimLeft(key, "/")

	object := s.container.Object(key)

	//calculate md5 hash
	md5Hash := md5.New()
	if _, err := io.Copy(md5Hash, tmpfile); err != nil {
		return nil, err
	}
	md5String := hex.EncodeToString(md5Hash.Sum(nil))
	//rewind reader
	if _, err := tmpfile.Seek(0, 0); err != nil {
		return nil, err
	}

	hdr := schwift.NewObjectHeaders()
	hdr.Etag().Set(md5String)

	if err := object.Upload(tmpfile, nil, hdr.ToOpts()); err != nil {
		return nil, err
	}
	objectURL, err := object.URL()
	if err != nil {
		return nil, err
	}
	return url.Parse(objectURL)

}
