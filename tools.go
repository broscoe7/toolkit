package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. Any variable
// of this type will have access to all the methods with the receiver *Tools.
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// RandomString returns a string of random characters of length n,
// drawing characters from randomStringSource.
func (t *Tools) RandomString(n int) string {
	if n <= 0 {
		return ""
	}
	// create slice of bytes and fill it with random byte values
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	// use bitwise operation to turn random byte into an index between 0 and 63
	for i := range buf {
		buf[i] = randomStringSource[buf[i]&63]
	}
	return string(buf)
}

// UploadedFile is a struct used to save information about an uploaded file
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// UploadOneFile is a convenience method that calls UploadFiles, but expects
// only one file in the upload.
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}
	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}
	return files[0], nil
}

// UploadFiles uploads one or more files to a specified directory, and gives
// the files a new randomized name. It returns a slice containing the newly named files,
// the original file, names, the size of the file, and potentially an error. If the optional
// rename parameter is set to false, the files will not be renamed.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	// in case MaxFileSize not set, provide a default value
	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	var uploadedFiles []*UploadedFile
	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	// create uploadDir if it does not exist
	err = t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				infile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				// examine the first 512 bytes to determine file type
				// more reliable than file extension
				buff := make([]byte, 512)
				_, err = infile.Read(buff)
				if err != nil {
					return nil, err
				}
				filetype := http.DetectContentType(buff)

				// check if file type is allowed
				var allowed bool
				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(filetype, x) {
							allowed = true
							break
						}
					}
				} else {
					allowed = true
				}
				if !allowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				// move file reader back to beginning of file
				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				// rename file for better security
				uploadedFile.OriginalFileName = hdr.Filename
				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(20), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}

				// store the file
				var outfile *os.File
				defer outfile.Close()
				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}
				uploadedFiles = append(uploadedFiles, &uploadedFile)
				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}
	return uploadedFiles, nil
}

// CreateDirIfNotExist creates a directory, together with any necessary parents, if it does not exist.
func (t *Tools) CreateDirIfNotExist(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
