package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("random string is wrong length")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "without rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
	{
		name:          "with rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "not allowed",
		allowedTypes:  []string{"image/jpeg"},
		renameFile:    false,
		errorExpected: true,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// create the form data field "file"
			filename := filepath.Join(".", "testdata", "img.png")
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				t.Error(err)
			}
			f, err := os.Open(filename)
			if err != nil {
				t.Error(err)
			}
			defer f.Close()
			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read data from the pipe and create an http request
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		// run test
		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(request, filepath.Join(".", "testdata", "uploads"), e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			uploadedPath := filepath.Join(".", "testdata", "uploads", uploadedFiles[0].NewFileName)
			if _, err = os.Stat(uploadedPath); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", uploadedPath, err.Error())
			}
			// cleanup
			_ = os.Remove(uploadedPath)
		}

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", e.name)
		}
		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// create the form data field "file"
		filename := filepath.Join(".", "testdata", "img.png")
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Error(err)
		}
		f, err := os.Open(filename)
		if err != nil {
			t.Error(err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read data from the pipe and create an http request
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	// run test
	var testTools Tools
	uploadedFile, err := testTools.UploadOneFile(request, filepath.Join(".", "testdata", "uploads"), true)
	if err != nil {
		t.Error(err)
	}
	uploadedPath := filepath.Join(".", "testdata", "uploads", uploadedFile.NewFileName)
	if _, err := os.Stat(uploadedPath); os.IsNotExist(err) {
		t.Errorf("%s: expected file to exist: %s", uploadedPath, err.Error())
	}
	// cleanup
	_ = os.Remove(uploadedPath)
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	path := filepath.Join(".", "testdata", "test-dir")
	var testTools Tools
	err := testTools.CreateDirIfNotExist(path)
	if err != nil {
		t.Error(err)
	}
	err = testTools.CreateDirIfNotExist(path)
	if err != nil {
		t.Error(err)
	}
	_ = os.Remove(path)
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "Now is the 123 #Time", expected: "now-is-the-123-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "japanese string", s: "こんにちは、お元気ですか?", expected: "", errorExpected: true},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools
	for _, e := range slugTests {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}
		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned - expected %s but got %s", e.name, e.expected, slug)
		}
		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", e.name)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools
	testTools.DownloadStaticFile(rr, req, "testdata", "pic.jpg", "puppy.jpg")

	result := rr.Result()
	defer result.Body.Close()

	// check size of response content
	if result.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content length of ", result.Header["Content-Length"][0])
	}

	// check that header has been appropriately set
	if result.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Error("wrong content disposition header")
	}

	// check that response body can be read
	_, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "bad json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "bar"}{"alpha": "beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json syntax error", json: `{"foo": bar"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field", json: `{"food": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown field", json: `{"food": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `Hello world`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools
	for _, e := range jsonTests {
		testTools.AllowUnknownFields = e.allowUnknown
		testTools.MaxJSONSize = e.maxSize
		var decodedJSON struct {
			Foo string `json:"foo"`
		}
		// create request with body (to pass in as "r")
		request, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error: ", err)
		}
		defer request.Body.Close()
		// create a recorder (to pass in as "w")
		rr := httptest.NewRecorder()
		err = testTools.ReadJSON(rr, request, &decodedJSON)

		// test cases
		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", e.name)
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: unexpected error received - %s", e.name, err.Error())
		}
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}
	headers := make(http.Header)
	headers.Add("Foo", "Bar")
	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding json response", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, but should be true")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	var testTools Tools
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"
	_, _, err := testTools.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url: ", err)
	}
}
