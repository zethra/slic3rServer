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
	"errors"
	"io/ioutil"
)

func TestSetUp(test *testing.T) {
	if err := SetUp(); err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}
	if err := CleanUp(); err != nil {
		test.Error(err)
	}
}

func TestFileList(test *testing.T) {
	if err := SetUp(); err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
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
	test.Logf("Recieved: %v\n", writer.Body)
	if err := CleanUp(); err != nil {
		test.Error(err)
	}
}

func TestDeleteFile(test *testing.T) {
	if err := SetUp(); err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}

	if _, err := os.Create("./stl/test.stl"); err != nil {
		test.Errorf("Failed to creat test stl file: %v", err)
	}
	request, _ := http.NewRequest("DELETE", "", nil)
	writer := httptest.NewRecorder()
	_getVars := getVars
	revert := func() {
		getVars = _getVars
	}
	defer revert()
	getVars = func(_ *http.Request) map[string]string {
		return map[string]string{
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

	if _, err := os.Create("./gcode/test.gcode"); err != nil {
		test.Errorf("Failed to creat test gcode file: %v", err)
	}
	request, _ = http.NewRequest("DELETE", "", nil)
	writer = httptest.NewRecorder()
	getVars = func(_ *http.Request) map[string]string {
		return map[string]string{
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
	if err := CleanUp(); err != nil {
		test.Error(err)
	}
}

func TestClearFiles(test *testing.T) {
	if err := SetUp(); err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}

	if _, err := os.Create("./stl/test.stl"); err != nil {
		test.Errorf("Failed to creat test stl file: %v", err)
	}
	if _, err := os.Create("./stl/test2.stl"); err != nil {
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

	if _, err := os.Create("./gcode/test.gcode"); err != nil {
		test.Errorf("Failed to creat test gcode file: %v", err)
	}
	if _, err := os.Create("./gcode/test2.gcode"); err != nil {
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
	if err := CleanUp(); err != nil {
		test.Error(err)
	}
}

type CallBackRequest struct {
	formData map[string]string
	callBackType string
}

type CallbackHandler struct {
	sync.Mutex
	callBackType string
	test *testing.T
}

func (handler *CallbackHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(writer, "Request is not a POST request", 400)
		return
	}
	handler.Lock()
	defer handler.Unlock()
	if handler.callBackType == "url" {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, "Could not read call back reuqest body", http.StatusInternalServerError)
			handler.test.Error("Could not read call back reuqest body")
		}
		if _, err := os.Stat("." + string(body)); err == nil {
			writer.WriteHeader(http.StatusOK)
			handler.test.Log("URL callback recieved")
		} else {
			http.Error(writer, "Gcode file was not created", http.StatusInternalServerError)
			handler.test.Error("Gcode file was not created")
		}
	} else if handler.callBackType == "file" {
		if err := request.ParseMultipartForm(32 << 20); err != nil {
			http.Error(writer, "Failed to parse callback multipart form", 500)
			handler.test.Error("Failed to parse callback multipart form")
		}
		_, header, err := request.FormFile("file")
		if err != nil {
			http.Error(writer, "Could not parse file form callback", 400)
			handler.test.Error("Could not parse file form callback")
		}
		if _, err := os.Stat("gcode/" + header.Filename); err == nil {
			writer.WriteHeader(http.StatusOK)
			handler.test.Log("file callback recieved")
		} else {
			http.Error(writer, "Gcode file was not created", http.StatusInternalServerError)
			handler.test.Error("Gcode file was not created")
		}
	}
}

func TestServerTest(test *testing.T) {
	if err := SetUp(); err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}
	handler := &CallbackHandler{callBackType:"url"}
	server := httptest.NewServer(handler)
	defer server.Close()
}

func TestSlicerFile(test *testing.T) {
	if err := SetUp(); err != nil {
		test.Fatalf("Setup failed with error: %v", err.Error())
	}

	handler := &CallbackHandler{test: test}
	server := httptest.NewServer(handler)
	defer server.Close()

	tests := [...]CallBackRequest{
		CallBackRequest{
			formData: map[string]string{},
			callBackType: "",
		},
		CallBackRequest{
			formData: map[string]string{
				"wait": "true",
			},
			callBackType: "",
		},
		CallBackRequest{
			formData: map[string]string{
				"callback": "url," + server.URL,
			},
			callBackType: "url",
		},
		CallBackRequest{
			formData: map[string]string{
				"callback": "file," + server.URL,
			},
			callBackType: "file",
		},
		CallBackRequest{
			formData: map[string]string{
				"callback": "url," + server.URL,
				"wait": "true",
			},
			callBackType: "url",
		},
		CallBackRequest{
			formData: map[string]string{
				"callback": "file," + server.URL,
				"wait": "true",
			},
			callBackType: "file",
		},
	}

	for _,callBackRequest := range tests {
		if callBackRequest.callBackType != "" {
			handler.callBackType = callBackRequest.callBackType
		}
		request, err := newFileUploadRequest("/slice", callBackRequest.formData, "file", "cube.stl")
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

	if err := CleanUp(); err != nil {
		test.Error(err)
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

func CleanUp() error {
	if err := RemoveContents("stl"); err != nil {
		return errors.New("Fail to clear STL folder:" + err.Error())
	}
	if err := RemoveContents("gcode"); err != nil {
		return errors.New("Fail to clear Gcode folder:" + err.Error())
	}
	return nil
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
