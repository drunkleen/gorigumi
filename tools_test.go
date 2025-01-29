package toolkit

import (
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"testing"
)

// TestTools_GenerateRandomString tests the GenerateRandomString method by generating a random
// string of length 10 and checking its length. It also tests that the generated strings are
// different by generating two strings and comparing them.
func TestTools_GenerateRandomString(t *testing.T) {
	var testTools Tools
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

			// crate the form data field
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

		var testTools Tools
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

		// crate the form data field
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

	var testTools Tools

	testTools.AllowedFileTypes = []string{"image/png"}

	UploadedSingleFile, err := testTools.UploadFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat("./testdata/uploads/" + UploadedSingleFile.NewFileName); os.IsNotExist(err) {
		t.Error("expected file to be created:", err.Error())
	}

	_ = os.Remove("./testdata/uploads/" + UploadedSingleFile.NewFileName)

}
