package main

import (
	"testing"
	"net/http"
	"net/http/httptest"
	//"encoding/json"
	"os"
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
		test.Errorf("File list didn't return OK, returned: %v", writer.Code)
	}
	/*var fileList []string
	if err = json.Unmarshal(writer.Body.Bytes(), fileList); err != nil {
		test.Errorf("File list didn't return valid json, returned: %v", writer.Body.String())
	}*/

	request, _ = http.NewRequest("GET", "/gcode", nil)
	writer = httptest.NewRecorder()
	fileListHandler(writer, request)
	if writer.Code != http.StatusOK {
		test.Errorf("File list didn't return OK, returned: %v", writer.Code)
	}
	/*if err = json.Unmarshal(writer.Body.Bytes(), fileList); err != nil {
		test.Errorf("File list didn't return valid json, returned: %v", writer.Body.String())
	}*/
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
		test.Errorf("File list didn't return NoContent, returned: %v", writer.Code)
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
		test.Errorf("File list didn't return NoContent, returned: %v", writer.Code)
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
		test.Errorf("File list didn't return NoContent, returned: %v", writer.Code)
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
		test.Errorf("File list didn't return NoContent, returned: %v", writer.Code)
	}
	if _, err := os.Stat("./gcode/test.gcode"); err == nil {
		test.Error("Gcode file was not deleted")
	}
	if _, err := os.Stat("./gcode/test2.gcode"); err == nil {
		test.Error("Gcode file was not deleted")
	}
}