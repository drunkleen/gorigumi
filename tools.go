package toolkit

import (
	"crypto/rand"
	"encoding/json"
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
	// MaxFileSize is the maximum file size in bytes
	MaxFileSize int
	// AllowedFileTypes is the list of allowed file types. Included '*'
	// indicates that all file types are allowed
	AllowedFileTypes []string
	// MaxJSONSize is the maximum size of a JSON object. Default to 1MB
	MaxJSONSize int
	// AllowUnknownFields is a boolean that indicates if unknown fields
	// are allowed in JSON
	AllowUnknownFields bool
}

// New returns a new empty instance of Tools.
func New() *Tools {
	return &Tools{}
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

	if err := t.CreateDirIfNotExists(uploadDir); err != nil {
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

	if err := t.CreateDirIfNotExists(uploadDir); err != nil {
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
		for _, v := range t.AllowedFileTypes {
			if strings.EqualFold(v, fileType) || strings.EqualFold(v, "*") {
				allowed = true
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

// DownloadFile sends a file to the client as an attachment.
// It takes four parameters, a http.ResponseWriter, a *http.Request, the path to the file,
// the filename of the file, and the name that the file should have when the client downloads it.
// The method sets the Content-Disposition header so that the file is downloaded as an attachment.
// It then uses http.ServeFile to send the file to the client.
func (t *Tools) DownloadFile(
	w http.ResponseWriter, r *http.Request,
	path, fileName, name string,
) {
	filePath := filepath.Join(path, fileName)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))

	http.ServeFile(w, r, filePath)
}

// CreateDirIfNotExists creates a directory if it does not exist.
// It takes a single parameter, the path to the directory.
// The method returns an error if the directory cannot be created.
func (t *Tools) CreateDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// ConvertToSlug converts a given string into a URL-friendly slug.
// It replaces all non-alphanumeric characters with hyphens and
// trims leading and trailing hyphens. The function returns an
// error if the input string is empty or if the resulting slug
// is empty due to invalid characters.
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

// JSONResponse is a struct that is used to return a JSON response to the client.
// It has three fields: Error, Message, and Data.
// The Error field is a boolean that indicates whether the response is an error or not.
// The Message field is a string that contains the error message if the response is an error.
// The Data field is a generic type that contains the data to be returned in the response.
// If the Data field is not set, it will be set to nil.
type JSONResponse struct {
	Error   bool   `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// ReadJSON reads a JSON request body into the given destination.
//
// The maximum size of the request body is set to 1MB by default, but can be
// overridden by setting the MaxJSONSize field of the Tools struct to a non-zero value.
//
// If the request body is too large, an error will be returned with the message
// "body must not be larger than X bytes".
//
// If the request body contains invalid JSON, an error will be returned with a
// message describing the problem.
//
// If the request body contains unknown fields and the AllowUnknownFields field
// of the Tools struct is set to false, an error will be returned with a message
// describing the unknown field.
//
// If the request body contains more than one JSON value, an error will be returned
// with the message "body should'nt contain more than one json value".
func (t *Tools) JSONRead(w http.ResponseWriter, r *http.Request, jsonData any) error {
	maxBytes := 1024 * 1024 // 1MB
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	if !t.AllowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(jsonData); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at position %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains an invalid JSON type at position %d", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("body contains badly-formed JSON (at position %d)", invalidUnmarshalError)

		default:
			return err
		}
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body should'nt contain more than one json value")
	}

	return nil

}

// JSONWrite writes a JSON response to the client with the specified HTTP status code.
// It takes an optional set of HTTP headers to include in the response. The function
// marshals the provided data into JSON format and writes it to the response writer.
// If marshaling the data fails, or if writing to the response writer fails, it returns an error.

func (t *Tools) JSONWrite(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(out); err != nil {
		return err
	}
	return nil
}

// JSONError writes an error response to the client with the specified HTTP status code.
// It takes an error and an optional HTTP status code as parameters. If the status code
// is not provided, it defaults to 500 Internal Server Error. The function marshals
// the error into a JSONResponse and writes it to the response writer. If marshaling
// the error fails, or if writing to the response writer fails, it returns an error.
func (t *Tools) JSONError(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusInternalServerError
	if len(status) > 0 {
		statusCode = status[0]
	}

	var res JSONResponse
	res.Error = true
	res.Message = err.Error()

	return t.JSONWrite(w, statusCode, res)
}
