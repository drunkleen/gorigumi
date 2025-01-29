package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// randomStringSource is used to generate random strings
	// it is inlcuded in the GenerateRandomString method
	randomStringSource string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

	// defaultMaxFileSize is the default maximum file size in bytes
	// it is inlcuded in the UploadFiles method
	defaultMaxFileSize int = 512 * 1024 * 1024 // default to 512MB
)

// Tools is the type used to instantiate this module.
// Any variable of this type will have access to all methods with receiver *Tools
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// GenerateRandomString generates a random string of length n.
// The string is composed of characters from the predefined
// randomStringSource, which includes uppercase and lowercase
// letters, digits, and an underscore.
func (t *Tools) GenerateRandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile struct represents an uploaded file.
// It contains the original file name, the new file name, and the file size.
type UploadedFile struct {
	OriginalFileName string
	NewFileName      string
	FileSize         int64
}

// UploadFiles parses a request and uploads all files in the request to the
// directory specified by uploadDir. It takes an optional boolean argument
// rename, which, if true, will rename all uploaded files with a random filename.
// The default value of rename is true. If MaxFileSize is not specified in the
// Tools struct, the default value of 512MB is used.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = defaultMaxFileSize
	}

	if err := t.CrateDirIfNotExists(uploadDir); err != nil {
		return nil, err
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded files are too big")
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFile, err := t.uploadCheck(hdr, uploadDir, renameFile)
			if err != nil {
				return uploadedFiles, err
			}
			uploadedFiles = append(uploadedFiles, uploadedFile)
		}
	}

	return uploadedFiles, nil
}

// UploadFile handles the upload of a single file from an HTTP request to the specified directory.
// If the optional rename argument is true or not provided, the uploaded file is renamed with a
// randomly generated filename. The function returns the details of the uploaded file or an error
// if the upload fails. It enforces the maximum file size defined in the Tools struct or defaults
// to 512MB if not specified.

func (t *Tools) UploadFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFile *UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = defaultMaxFileSize
	}

	if err := t.CrateDirIfNotExists(uploadDir); err != nil {
		return nil, err
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	for _, fileHeader := range r.MultipartForm.File {
		uploadedFile, err = t.uploadCheck(fileHeader[0], uploadDir, renameFile)
		if err != nil {
			return uploadedFile, err
		}
	}

	return uploadedFile, nil

}

// uploadCheck parses a single file from an HTTP request and uploads it to the directory
// specified by uploadDir. If the optional rename argument is true or not provided, the
// uploaded file is renamed with a randomly generated filename. The function returns the
// details of the uploaded file or an error if the upload fails. It enforces the maximum
// file size defined in the Tools struct or defaults to 512MB if not specified.
func (t *Tools) uploadCheck(
	hdr *multipart.FileHeader, uploadDir string, renameFile bool,
) (*UploadedFile, error) {
	var file UploadedFile

	inFile, err := hdr.Open()

	if err != nil {
		return nil, err
	}
	defer inFile.Close()

	buff := make([]byte, 512)
	_, err = inFile.Read(buff)
	if err != nil {
		return nil, err
	}

	allowed := false
	fileType := http.DetectContentType(buff)

	if len(t.AllowedFileTypes) > 0 {
		if t.AllowedFileTypes[0] == "*" {
			allowed = true
		} else {
			for _, v := range t.AllowedFileTypes {
				if strings.EqualFold(v, fileType) {
					allowed = true
				}
			}
		}
	}

	if !allowed {
		return nil, errors.New("file type is not allowed")
	}

	_, err = inFile.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	if renameFile {
		file.NewFileName = fmt.Sprintf("%s_%s", t.GenerateRandomString(32), filepath.Ext(hdr.Filename))
	} else {
		file.NewFileName = hdr.Filename
	}

	file.OriginalFileName = hdr.Filename

	var oFile *os.File
	defer oFile.Close()

	if oFile, err = os.Create(filepath.Join(uploadDir, file.NewFileName)); err != nil {
		return nil, err
	} else {
		fileSize, err := io.Copy(oFile, inFile)
		if err != nil {
			return nil, err
		}
		file.FileSize = fileSize
	}

	return &file, nil

}

// CrateDirIfNotExists creates the directory at the given path if it does not
// already exist. If the directory does exist, this method does nothing and
// returns nil. If the directory does not exist, this method creates it with
// permission 0755 and returns nil if successful, or an error if the
// operation fails.
func (t *Tools) CrateDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// ConvertToSlug converts a given string to a slug string.
// It returns the slug string and an error if the string is empty or only contains
// invalid characters.
func (t *Tools) ConvertToSlug(s string) (string, error) {
	if s == "" {
		return "", errors.New("string is empty")
	}
	regex := regexp.MustCompile(`[^a-z\d]+`)

	slug := strings.Trim(
		regex.ReplaceAllString(strings.ToLower(s), "-"), "-",
	)

	if len(slug) == 0 {
		return "", errors.New("not valid string characters")
	}

	return slug, nil
}
