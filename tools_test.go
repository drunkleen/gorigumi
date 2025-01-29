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
	os.RemoveAll("./testdata/uploads/")

}

// TestTools_CrateDirIfNotExists tests the CrateDirIfNotExists method by creating a directory,
// then trying to create it again. The test checks that the first call succeeds and the
// second call does nothing and returns nil.
func TestTools_CrateDirIfNotExists(t *testing.T) {
	var testTools Tools
	if err := testTools.CrateDirIfNotExists("./testdata/test-dir"); err != nil {
		t.Error(err)
	}

	if err := testTools.CrateDirIfNotExists("./testdata/test-dir"); err != nil {
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
	var testTools Tools

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
