package main

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"encoding/json"
)

func TestSetUp(t *testing.T)  {
	err := SetUp()
	if(err != nil) {
		t.Fatalf("Setup fialed with error: %v", err.Error())
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
		test.Errorf("File list didn't return OK, returned: %v", writer.Code)
	}
	if err = json.Unmarshal(writer.Body.Bytes(), &config); err != nil {
		test.Error("File list didn't return valid json")
	}
}