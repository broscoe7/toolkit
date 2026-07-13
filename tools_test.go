package toolkit

import (
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
