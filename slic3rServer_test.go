package main

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"mime/multipart"
	"bytes"
	"path/filepath"
	"io"
	"time"
)

func TestSetUp(t *testing.T)  {
	err := SetUp()
	if(err != nil) {
		t.Fatalf("Setup failed with error: %v", err.Error())
	}
}

func TestFileList(test *testing.T) {
	err := SetUp()
	if(err != nil) {
		test.Fatalf("Setup fialed with error: %v", err.Error())
	}
	request, _ := http.NewRequest("GET", "/stl", nil)
	writer := httptest.NewRecorder()
	fileListHandler(writer, request)
	if writer.Code != http.StatusOK {
		test.Errorf("File list didn't return OK, returned: %v - %v", writer.Code, writer.Body.String())
	}

	request, _ = http.NewRequest("GET", "/gcode", nil)
	writer = httptest.NewRecorder()
	fileListHandler(writer, request)
	if writer.Code != http.StatusOK {
		test.Errorf("File list didn't return OK, returned: %v - %v", writer.Code, writer.Body.String())
	}
}

func TestDeleteFile(test *testing.T) {
	err := SetUp()
	if err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}

	_, err = os.Create("./stl/test.stl")
	if err != nil {
		test.Errorf("Failed to creat test stl file: %v", err)
	}
	request, _ := http.NewRequest("DELETE", "", nil)
	writer := httptest.NewRecorder()
	_getVars := getVars
	getVars = func(_ *http.Request) map[string]string{
		return map[string]string {
			"type": "stl",
			"name": "test.stl",
		}
	}
	deleteFileHandler(writer, request)
	if writer.Code != http.StatusNoContent {
		test.Errorf("File list didn't return NoContent, returned: %v - %v", writer.Code, writer.Body.String())
	}
	if _, err := os.Stat("./stl/test.stl"); err == nil {
		test.Error("STL file was not deleted")
	}

	_, err = os.Create("./gcode/test.gcode")
	if err != nil {
		test.Errorf("Failed to creat test gcode file: %v", err)
	}
	request, _ = http.NewRequest("DELETE", "", nil)
	writer = httptest.NewRecorder()
	getVars = func(_ *http.Request) map[string]string{
		return map[string]string {
			"type": "gcode",
			"name": "test.gcode",
		}
	}
	deleteFileHandler(writer, request)
	if writer.Code != http.StatusNoContent {
		test.Errorf("File list didn't return NoContent, returned: %v - %v", writer.Code, writer.Body.String())
	}
	if _, err := os.Stat("./gcode/test.gcode"); err == nil {
		test.Error("Gcode file was not deleted")
	}
	getVars = _getVars
}

func TestClearFiles(test *testing.T) {
	err := SetUp()
	if err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}

	_, err = os.Create("./stl/test.stl")
	if err != nil {
		test.Errorf("Failed to creat test stl file: %v", err)
	}
	_, err = os.Create("./stl/test2.stl")
	if err != nil {
		test.Errorf("Failed to creat test stl file: %v", err)
	}
	request, _ := http.NewRequest("DELETE", "/stl", nil)
	writer := httptest.NewRecorder()
	clearFilesHandler(writer, request)
	if writer.Code != http.StatusNoContent {
		test.Errorf("File list didn't return NoContent, returned: %v - %v", writer.Code, writer.Body.String())
	}
	if _, err := os.Stat("./stl/test.stl"); err == nil {
		test.Error("STL file was not deleted")
	}
	if _, err := os.Stat("./stl/test2.stl"); err == nil {
		test.Error("STL file was not deleted")
	}

	_, err = os.Create("./gcode/test.gcode")
	if err != nil {
		test.Errorf("Failed to creat test gcode file: %v", err)
	}
	_, err = os.Create("./gcode/test2.gcode")
	if err != nil {
		test.Errorf("Failed to creat test gcode file: %v", err)
	}
	request, _ = http.NewRequest("DELETE", "/gcode", nil)
	writer = httptest.NewRecorder()
	clearFilesHandler(writer, request)
	if writer.Code != http.StatusNoContent {
		test.Errorf("File list didn't return NoContent, returned: %v - %v", writer.Code, writer.Body.String())
	}
	if _, err := os.Stat("./gcode/test.gcode"); err == nil {
		test.Error("Gcode file was not deleted")
	}
	if _, err := os.Stat("./gcode/test2.gcode"); err == nil {
		test.Error("Gcode file was not deleted")
	}
	err = os.Remove("stl")
}

func TestSlicerFile(test *testing.T) {
	type CallbackHandler struct {
		sync.Mutex
		code int
		body string

	}
	err := SetUp()
	if err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}

	_, err = os.Create("./stl/test.stl")
	if err != nil {
		test.Errorf("Failed to creat test stl file: %v", err)
	}
	formData := map[string]string{
		//"wait": "true",
	}
	request, err := newFileUploadRequest("/slice", formData, "file", "cube.stl")
	if err != nil {
		test.Errorf("Failed to creat multipart request: %v", err.Error())
	}
	writer := httptest.NewRecorder()
	sliceHandler(writer, request)
	time.Sleep(1 * time.Second)
	if writer.Code != http.StatusOK {
		test.Errorf("File list didn't return OK, returned: %v - %v", writer.Code, writer.Body.String())
	}
	if _, err := os.Stat("stl/cube.stl"); os.IsNotExist(err) {
		test.Error("STL file wasn't uploaded")
	}
	if _, err := os.Stat("gcode/cube.gcode"); os.IsNotExist(err) {
		test.Error("Gcode file wasn't created")
	}
}

func newFileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request, nil
}
