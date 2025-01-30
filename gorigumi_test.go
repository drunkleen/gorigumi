package gorigumi

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestTools_GenerateRandomString tests the GenerateRandomString method by generating a random
// string of length 10 and checking its length. It also tests that the generated strings are
// different by generating two strings and comparing them.
func TestTools_GenerateRandomString(t *testing.T) {
	testTools := New()
	randString := testTools.GenerateRandomString(10)
	if len(randString) != 10 {
		t.Error("Expected string of length 10, but got", len(randString))
	}
	randString1, randString2 := testTools.GenerateRandomString(10), testTools.GenerateRandomString(10)
	if randString1 == randString2 {
		t.Error("Expected random strings to be different")
	}

}

// uploadTests is a slice of structs that hold the name of the test, the allowed file types, a boolean
// that indicates if the file should be renamed, and a boolean that indicates if an error is expected
var uploadTests = []struct {
	name            string
	alowedFileTypes []string
	renameFile      bool
	errorExpected   bool
}{
	{"allowed no rename", []string{"image/jpeg", "image/png"}, false, false},
	{"allowed rename", []string{"image/jpeg", "image/png"}, true, false},
	{"not allowed", []string{"image/jpeg"}, false, true},
}

// TestTools_uploadFiles tests the UploadFiles method by simulating a request with a single file in the form data.
// The file is read from testdata/img.png and written to the pipe. The AllowedFileTypes is set to only allow PNG files.
// The test checks that the file is uploaded and that the error returned is nil.
func TestTools_uploadFiles(t *testing.T) {
	for _, entry := range uploadTests {
		pipeReader, pipeWriter := io.Pipe()
		writer := multipart.NewWriter(pipeWriter)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer writer.Close()

			// create the form data field
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("Error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe
		request, _ := http.NewRequest("POST", "/", pipeReader)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		testTools := New()
		testTools.AllowedFileTypes = entry.alowedFileTypes

		UploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", entry.renameFile)
		if err != nil && !entry.errorExpected {
			t.Error(err)
		}

		if !entry.errorExpected {
			if _, err := os.Stat("./testdata/uploads/" + UploadedFiles[0].NewFileName); os.IsNotExist(err) {
				t.Errorf("%s: expected file to be created: %s", entry.name, err.Error())
			}
			_ = os.Remove("./testdata/uploads/" + UploadedFiles[0].NewFileName)
		}

		if entry.errorExpected && err == nil {
			t.Errorf("%s: expected error to be returned but got none", entry.name)
		}

		wg.Wait()
	}
	os.RemoveAll("./testdata/uploads/")
}

// TestTools_uploadSingleFile tests the UploadFile method by simulating a request
// with a single file in the form data. The file is read from testdata/img.png
// and written to the pipe. The AllowedFileTypes is set to only allow PNG files.
// The test checks that the file is uploaded and that the error returned is nil.
func TestTools_uploadSingleFile(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)

	go func() {
		defer writer.Close()

		// create the form data field
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("Error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe
	request, _ := http.NewRequest("POST", "/", pipeReader)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	testTools := New()

	testTools.AllowedFileTypes = []string{"image/png"}

	UploadedSingleFile, err := testTools.UploadFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat("./testdata/uploads/" + UploadedSingleFile.NewFileName); os.IsNotExist(err) {
		t.Error("expected file to be created:", err.Error())
	}

	_ = os.Remove("./testdata/uploads/" + UploadedSingleFile.NewFileName)
	os.RemoveAll("./testdata/uploads/")

}

func TestTools_DownloadFile(t *testing.T) {
	testTools := New()
	responseRecorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	testTools.DownloadFile(responseRecorder, req, "./testdata", "img.png", "rgb.png")

	if responseRecorder.Code != http.StatusOK {
		t.Error("Expected status code 200, but got", responseRecorder.Code)
	}

	result := responseRecorder.Result()
	result.Body.Close()

	if result.Header.Get("Content-Type") != "image/png" {
		t.Error("Expected content-type image/png, but got", result.Header.Get("Content-Type"))
	}

	if result.Header.Get("Content-Disposition") != "attachment; filename=\"rgb.png\"" {
		t.Error("Expected content-disposition attachment; filename=\"rgb.png\", but got", result.Header.Get("Content-Disposition"))
	}

	if result.Header.Get("Content-Length") != "1422" {
		t.Error("Expected content-length 100, but got", result.Header.Get("Content-Length"))
	}

	if _, err := io.ReadAll(result.Body); err != nil {
		t.Error(err)
	}
}

// TestTools_CreateDirIfNotExists tests the CreateDirIfNotExists method by creating a directory,
// then trying to create it again. The test checks that the first call succeeds and the
// second call does nothing and returns nil.
func TestTools_CreateDirIfNotExists(t *testing.T) {
	testTools := New()
	if err := testTools.CreateDirIfNotExists("./testdata/test-dir"); err != nil {
		t.Error(err)
	}

	if err := testTools.CreateDirIfNotExists("./testdata/test-dir"); err != nil {
		t.Error(err)
	}

	os.RemoveAll("./testdata/test-dir")
}

var slugTests = []struct {
	name          string
	input         string
	expected      string
	errorExpected bool
}{
	{"valid string", "Hello, World!", "hello-world", false},
	{"valid number", "12345 67890", "12345-67890", false},
	{"valid string with non-english characters", "Hello Word!你好世界", "hello-word", false},
	{"valid string with numbers and special characters", "Hello, World! 1234567890 !@#$%^&*()", "hello-world-1234567890", false},
	{"empty string", "", "", true},
	{"invalid number", "!@#$%^&*()", "", true},
	{"invalid persian characters", "سلام دنیا", "", true},
	{"invalid chinese characters", "你好世界", "", true},
	{"invalid japanese characters", "こんにちは世界", "", true},
}

// TestTools_ConvertToSlug tests the ConvertToSlug method by converting valid strings to their
// slug form and checking that invalid strings return an error. It also tests that the method
// correctly handles strings with special characters and non-english characters.
func TestTools_ConvertToSlug(t *testing.T) {
	testTools := New()

	for _, st := range slugTests {
		s, err := testTools.ConvertToSlug(st.input)

		if err != nil && st.errorExpected {
			continue
		}
		if err == nil && st.errorExpected {
			t.Errorf("%s: expected error but got none", st.name)
			continue
		}
		if err != nil && !st.errorExpected {
			t.Errorf("%s: %s`", st.name, err)
			continue
		}
		if s != st.expected && !st.errorExpected {
			t.Errorf("%s: expected %s, got %s", st.name, st.expected, s)
			continue
		}
	}

}

// JSONTests is a slice of structs that hold the name of the test, the input JSON string,
// the maximum size of the JSON file, and a boolean that indicates if an error is expected
var JSONTests = []struct {
	name               string
	inputJSON          string
	maxSize            int
	allowUnknownFields bool
	errorExpected      bool
}{
	{"valid JSON", `{"str": "foo"}`, 1024, false, false},
	{"badly-formed JSON", `{"str":}`, 1024, false, true},
	{"incorrect type JSON", `{"str": 42}`, 1024, false, true},
	{"double JSON data", `{"str": "foo"}{"str": "foo"}`, 1024, false, true},
	{"unkown field key", `{"str": "foo", "num": 42}`, 1024, false, true},
	{"allow unkown field key", `{"str": "foo", "num": 42}`, 1024, true, false},
	{"JSON syntax error", `{"str" "foo"}`, 1024, false, true},
	{"missing closing brace", `{"str" "foo"`, 1024, false, true},
	{"missing opening brace", `"str" "foo"}`, 1024, false, true},
	{"not JSON", `test string`, 1024, false, true},
	{"large JSON size", `"str" "foo"}`, 4, false, true},
}

// TestTools_JSONRead tests the JSONRead method of the Tools struct by sending various JSON
// payloads in HTTP requests and checking for expected outcomes. It verifies that the method
// correctly handles different scenarios such as valid JSON, malformed JSON, unknown fields,
// and size constraints. The test iterates over a set of predefined test cases, adjusting the
// MaxJSONSize and AllowUnknownFields settings for each case, and checks that the resulting
// behavior matches the expected error state and HTTP response status code.

func TestTools_JSONRead(t *testing.T) {
	testTools := New()

	for _, jt := range JSONTests {
		testTools.MaxJSONSize = jt.maxSize
		testTools.AllowUnknownFields = jt.allowUnknownFields

		var decodedJSON struct {
			Str string `json:"str"`
		}

		req, err := http.NewRequest("POST", "/", strings.NewReader(jt.inputJSON))
		if err != nil {
			t.Error(err)
		}
		defer req.Body.Close()

		responseRecorder := httptest.NewRecorder()

		err = testTools.JSONRead(responseRecorder, req, &decodedJSON)

		if jt.errorExpected && err == nil {
			t.Errorf("%s: Expected error but got none", jt.name)
		}

		if !jt.errorExpected && err != nil {
			t.Errorf("%s: %s", jt.name, err)
		}

		if responseRecorder.Code != http.StatusOK {
			t.Errorf("%s: expected status code %d, got %d", jt.name, http.StatusOK, responseRecorder.Code)
		}

		if _, err := io.ReadAll(responseRecorder.Body); err != nil {
			t.Error(err)
		}

	}
}

// TestTools_JSONWrite tests the JSONWrite method by writing a JSON response to the client
// with a status code of 200 and a JSON payload.
func TestTools_JSONWrite(t *testing.T) {
	testTools := New()

	responseRecorder := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("foo", "bar")

	if err := testTools.JSONWrite(responseRecorder, 200, payload); err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}

}

func TestTools_JSONError(t *testing.T) {
	testTools := New()

	responseRecorder := httptest.NewRecorder()

	if err := testTools.JSONError(responseRecorder, errors.New("foo"), http.StatusBadGateway); err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}

	var res JSONResponse
	decoder := json.NewDecoder(responseRecorder.Body)
	if err := decoder.Decode(&res); err != nil {
		t.Errorf("failed to decode JSON: %v", err)
	}

	if !res.Error {
		t.Errorf("expected error, but got success")
	}

	if responseRecorder.Code != http.StatusBadGateway {
		t.Errorf("expected status code %d, got %d", http.StatusBadGateway, responseRecorder.Code)
	}

	if res.Message != "foo" {
		t.Errorf("expected message 'foo', got %s", res.Message)
	}
}

type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip implements the RoundTripper interface. It simply calls the
// function passed to NewTestClient and returns the result as a *http.Response
// and a nil error.
func (r RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req), nil
}

// NewTestClient creates a new http.Client from a RoundTripFunc. The RoundTripFunc
// passed to NewTestClient is used as the Transport for the new http.Client.
//
// The returned http.Client can be used in tests to mock out the result of an HTTP
// request. The RoundTripFunc can return any *http.Response and error, allowing
// for complete control over the response.
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

// TestTools_JSONPushToRemote tests the JSONPushToRemote method by creating a test http client that always
// returns a 200 status code and a JSON payload. It then calls JSONPushToRemote with this client and a
// test struct, and checks that the method returns without error.
func TestTools_JSONPushToRemote(t *testing.T) {
	client := NewTestClient(
		func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("ok")),
				Header:     make(http.Header),
			}
		},
	)

	testTools := New()

	var foo struct {
		Bar string `json:"bar"`
	}

	foo.Bar = "BAR"

	if _, _, err := testTools.JSONPushToRemote("http//example.com/none/existing/path", foo, client); err != nil {
		t.Errorf("failed to reach remote url: %v", err)
	}
}
