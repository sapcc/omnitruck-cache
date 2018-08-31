package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/im7mortal/kmutex"
)

var (
	omnitruckURL     string
	cacheBackendName string
)

type OmnitruckProxy struct {
	Cache      Cache
	client     http.Client
	BackendURL string
	mutex      *kmutex.Kmutex
}

type OmnitruckResponse struct {
	Url     string `json:"url"`
	Sha256  string `json:"sha256"`
	Sha1    string `json:"sha1"`
	Version string `json:"version"`
}

func init() {
	flag.StringVar(&omnitruckURL, "omnitruck-url", "https://omnitruck.chef.io", "backend omnitruck url")
	flag.StringVar(&cacheBackendName, "cache-backend", "local", "which cache backend to use")
}

func main() {

	flag.Parse()

	var cacheBackend Cache
	switch cacheBackendName {
	case "local":
		localCache := &LocalCache{BaseURL: "http://localhost:8080/packages", CacheDir: "cache"}
		http.Handle("/packages/", http.StripPrefix("/packages/", http.FileServer(http.Dir(localCache.CacheDir))))
		cacheBackend = localCache
	case "swift":
		var err error
		cacheBackend, err = NewSwiftCache(os.Getenv("OS_CONTAINER"))
		if err != nil {
			log.Fatalf("Failed to initialize swift backend: %s", err)
		}
	default:
		log.Fatalf("unknown cache backend: %s", cacheBackendName)
	}

	log.Printf("Using %s cache backend", cacheBackendName)

	proxy := OmnitruckProxy{Cache: cacheBackend, BackendURL: omnitruckURL, mutex: kmutex.New()}
	proxy.client.Timeout = 15 * time.Minute
	http.HandleFunc("/health", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Write([]byte("ok"))
	})
	http.Handle("/", &proxy)

	listen := ":8080"
	log.Printf("Listening on %s", listen)
	log.Fatal(http.ListenAndServe(listen, handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)))
}

func (o *OmnitruckProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	response, err := o.proxy(req)

	if req.Header.Get("Accept") == "application/json" {
		rw.Header().Set("Content-Type", "application/json")
		if err == nil {
			json.NewEncoder(rw).Encode(response)

		} else {
			rw.WriteHeader(500)
			msg := map[string]string{"error": err.Error()}
			json.NewEncoder(rw).Encode(msg)
		}
	} else {
		rw.Header().Set("Content-Type", "text/plain")
		if err == nil {
			rw.Write([]byte(fmt.Sprintf(
				"sha1\t%s\nsha256\t%s\nurl\t%s\nversion\t%s\n",
				response.Sha1,
				response.Sha256,
				response.Url,
				response.Version,
			)))
		} else {
			log.Println(err)
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))
		}
	}

}

func (o *OmnitruckProxy) proxy(req *http.Request) (*OmnitruckResponse, error) {
	backendURL := fmt.Sprintf("%s%s", o.BackendURL, req.RequestURI)

	backendReq, err := http.NewRequest(req.Method, backendURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create backend request: %v", err)
	}
	backendReq.Header.Set("Accept", "application/json")

	response, err := o.client.Do(backendReq)
	if err != nil {
		return nil, fmt.Errorf("Failed to perform backend request: %v", err)
	}
	responseBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("Failed to read backend response: %v", err)
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("Backend responded with %s: %s", response.Status, []byte(responseBytes))
	}
	var omni OmnitruckResponse
	if err := json.Unmarshal(responseBytes, &omni); err != nil {
		return nil, fmt.Errorf("Failed to parse backend response. response: %s, error: %v", string(responseBytes), err)
	}

	packageURL, err := url.Parse(omni.Url)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse package url: %s, error: %v", omni.Url, err)
	}

	key := packageURL.EscapedPath()
	o.mutex.Lock(key)
	defer o.mutex.Unlock(key)
	cacheUrl, found, err := o.Cache.Get(packageURL.EscapedPath())
	if err != nil {
		return nil, fmt.Errorf("Failed to query the cache: %v", err)
	}
	if !found {
		log.Printf("Caching %s", omni.Url)
		resp, err := o.client.Get(omni.Url)
		if err != nil {
			return nil, fmt.Errorf("Fetching %s failed: %v", omni.Url, err)
		}
		defer resp.Body.Close()

		tmpfile, err := ioutil.TempFile("", "chef")
		if err != nil {
			return nil, fmt.Errorf("Unable to create temp file: %s", err)
		}

		defer os.Remove(tmpfile.Name())

		hash := sha256.New()
		multi := io.MultiWriter(tmpfile, hash)
		if _, err = io.Copy(multi, resp.Body); err != nil {
			return nil, fmt.Errorf("Failed to download %s: %s", omni.Url, err)
		}
		if err = tmpfile.Close(); err != nil {
			return nil, err
		}
		computedHash := hex.EncodeToString(hash.Sum(nil))
		if computedHash != omni.Sha256 {
			return nil, fmt.Errorf("Sha256 hash of downloaded file does not match. Expected %s, Got %s", omni.Sha256, computedHash)
		}

		cacheUrl, err = o.Cache.Store(packageURL.EscapedPath(), resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed to store %s in cache: %v ", omni.Url, err)
		}
	}
	omni.Url = cacheUrl.String()
	return &omni, nil
}
