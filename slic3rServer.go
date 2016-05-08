package main

import (
	"os"
	"log"
	"net/http"
	"io"
	"sync"
	"strings"
	"os/exec"
	"fmt"
	"encoding/xml"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"flag"
	"path/filepath"
	"mime/multipart"
	"net/url"

	"github.com/gorilla/mux"
)

type Config struct {
	Port       int `xml:"port"`
	Slic3rPath string `xml:"slic3rPath"`
}

var config = Config{}
//Declare flags
var debugFlag = flag.Bool("debug", false, "If set debug output will print")
var portFlag = flag.Int("port", 0, "If set slic3r server will bind to given port and will override config file")
var slic3rPathFlag = flag.String("", "slic3r", "If set slic3r server will use given sli3r path and will override config file")

func main() {
	log.Println("Starting Slic3r Server")
	//Parse flags
	flag.Parse()
	err := SetUp()
	if err != nil {
		panic(err)
	}
	server := NewServer()
	http.Handle("/", server)
	log.Printf("Slic3r Server binding to port: %d\n", config.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}

func SetUp() error{
	//Generate Directories
	if _, err := os.Stat("stl"); os.IsNotExist(err) {
		if (*debugFlag) {
			log.Println("Making STL Directory")
		}
		os.Mkdir("stl", 0777)
	}
	if _, err := os.Stat("gcode"); os.IsNotExist(err) {
		if *debugFlag {
			log.Println("Making Gcode Directory")
		}
		os.Mkdir("gcode", 0777)
	}
	//Create config file if does not exist
	if _, err := os.Stat("config.xml"); os.IsNotExist(err) {
		if *debugFlag {
			log.Println("Making config")
		}
		config.Port = 7766
		config.Slic3rPath = "slic3r"
		xml, err := xml.MarshalIndent(config, "", "    ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile("config.xml", xml, 0666)
		if err != nil {
			return err
		}
	} else {
		//Read config file if does exist
		data, err := ioutil.ReadFile("config.xml")
		if err != nil || string(data) == "" {
			return err
		}
		if err = xml.Unmarshal(data, &config); err != nil {
			return err
		}
		if *debugFlag {
			log.Println(config)
		}
	}
	//Override config with flags if set
	if *portFlag != 0 {
		config.Port = *portFlag
	}
	if *slic3rPathFlag != "" {
		config.Slic3rPath = *slic3rPathFlag
	}
	return nil
}

func NewServer() *mux.Router{
	//Start HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/slice", sliceHandler).Methods("POST")
	router.Handle("/gcode/{name}", http.StripPrefix("/gcode/", http.FileServer(http.Dir("gcode")))).Methods("GET")
	router.Handle("/stl/{name}", http.StripPrefix("/stl/", http.FileServer(http.Dir("stl")))).Methods("GET")
	router.HandleFunc("/{stl|gcode}", fileListHandler).Methods("GET")
	router.HandleFunc("/{stl|gcode}", clearFilesHandler).Methods("DELETE")
	router.HandleFunc("/{type:stl|gcode}/{name}", deleteFileHandler).Methods("DELETE")
	return router
}

func fileListHandler(writer http.ResponseWriter, request *http.Request) {
	files, err := ioutil.ReadDir("." + request.URL.String())
	if err != nil {
		http.Error(writer, "Could not get file list", 500)
		log.Println(err)
		return
	}
	var fileList []string
	for _, file := range files {
		fileList = append(fileList, file.Name())
	}
	data, err := json.MarshalIndent(fileList, "", "    ")
	if err != nil {
		http.Error(writer, "Could not get file list", 500)
		log.Println(err)
		return
	}
	writer.Write(data)
}

func deleteFileHandler(writer http.ResponseWriter, request *http.Request) {
	vars := getVars(request)
	fileType := vars["type"]
	fileName := vars["name"]
	if err := os.Remove("./" + fileType + "/" + fileName); err != nil {
		log.Println(err)
		http.Error(writer, "Failed to delete file", 500)
		return
	}
	writer.WriteHeader(204)
}

func clearFilesHandler(writer http.ResponseWriter, request *http.Request) {
	d, err := os.Open("." + request.URL.String())
	if err != nil {
		log.Println(err)
		http.Error(writer, "Failed to delete files", 500)
		return
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		log.Println(err)
		http.Error(writer, "Failed to delete files", 500)
		return
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join("." + request.URL.String(), name))
		if err != nil {
			log.Println(err)
			http.Error(writer, "Failed to delete files", 500)
			return
		}
	}
	writer.WriteHeader(204)
}

func sliceHandler(writer http.ResponseWriter, request *http.Request) {
	//Reject request if it is not a POST request
	if request.Method != "POST" {
		http.Error(writer, "Request is not a POST request", 400)
		return
	}
	//Get form data
	request.ParseMultipartForm(32 << 20)
	var otherArgs, callbackType, callbackURL string
	var wait bool
	for key, value := range request.Form {
		if key == "callback" && len(value) > 0 {
			tmp := strings.Split(value[0], ",")
			callbackType = tmp[0]
			callbackURL = tmp[1]
		} else if key == "wait" && len(value) > 0 {
			if value[0] == "true" {
				wait = true
			}
			if value[0] != "true" && value[0] != "false" {
				http.Error(writer, "Invalid value given for wait", 400)
				return
			}
		} else {
			if (len(value) > 0) {
				otherArgs += fmt.Sprintf(" --%s %s", key, value[0])
			} else {
				otherArgs += fmt.Sprintf(" --%s", key)
			}
		}
	}
	//Check if request is valid
	if (callbackType != "" && callbackType != "url" && callbackType != "file") {
		http.Error(writer, "Invalid callback type", 400)
		return
	}
	if (callbackURL != "") {
		_, err := url.Parse(callbackURL)
		if (err != nil) {
			http.Error(writer, "Invalid callback URL", 400)
			return
		}
	}
	//Get STL file
	tmpFile, header, err := request.FormFile("file")
	if err != nil {
		log.Println(err)
		http.Error(writer, "Could not parse file form request", 400)
		return
	}
	defer tmpFile.Close()
	fileName := header.Filename[:(len(header.Filename) - 4)]
	file, err := os.OpenFile("stl/" + header.Filename, os.O_WRONLY | os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
		http.Error(writer, "Could not open file: stl/" + header.Filename, 500)
		return
	}
	io.Copy(file, tmpFile)
	file.Close()
	//Run slic3r with STL file and args
	args := fmt.Sprintf(" stl/%s.stl %s --output gcode/%s.gcode", fileName, otherArgs, fileName)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go exe_cmd(config.Slic3rPath + args, wg)
	gcodeFile := "/gcode/" + fileName + ".gcode"
	//Wait if needed
	if (!wait) {
		writer.Write([]byte(gcodeFile))
	}else if (wait && callbackURL == "") {
		wg.Wait()
		writer.Write([]byte(gcodeFile))
	}
	//Run callback
	if (callbackType == "url" && callbackURL != "") {
		wg.Wait()
		req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer([]byte(gcodeFile)))
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		client := &http.Client{}
		_, err = client.Do(req)
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		if (wait) {
			writer.Write([]byte(gcodeFile))
		}
	} else if (callbackType == "file" && callbackURL != "") {
		wg.Wait()
		file, err := os.Open(gcodeFile[1:len(gcodeFile)])
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		defer file.Close()
		body := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(body)
		part, err := mpWriter.CreateFormFile("file", filepath.Base(gcodeFile[1:len(gcodeFile)]))
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		_, err = io.Copy(part, file)
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		err = mpWriter.Close()
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		req, err := http.NewRequest("POST", callbackURL, body)
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		client := &http.Client{}
		_, err = client.Do(req)
		if (err != nil) {
			log.Println(err)
			if (wait) {
				http.Error(writer, "Callback could not be completed", 500)
			}
			return
		}
		if (wait) {
			writer.Write([]byte(gcodeFile))
		}
	}
}

func exe_cmd(cmd string, wg *sync.WaitGroup) {
	if (*debugFlag) {
		log.Println("executing: ", cmd)
	}
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		log.Printf("%s\n", err)
	}
	if (*debugFlag) {
		log.Printf("%s\n", out)
	}
	wg.Done()
}

var getVars = func(request *http.Request) map[string]string {
	return mux.Vars(request)
}
