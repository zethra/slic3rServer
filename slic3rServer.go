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
	//Generate Directories
	if _, err := os.Stat("stl"); os.IsNotExist(err) {
		if (*debugFlag) {
			log.Println("Making STL Directory")
		}
		os.Mkdir("stl", 0777)
	}
	if _, err := os.Stat("gcode"); os.IsNotExist(err) {
		if (*debugFlag) {
			log.Println("Making Gcode Directory")
		}
		os.Mkdir("gcode", 0777)
	}
	//Create config file if does not exist
	if _, err := os.Stat("config.xml"); os.IsNotExist(err) {
		if (*debugFlag) {
			log.Println("Making config")
		}
		config.Port = 7766
		config.Slic3rPath = "slic3r"
		xml, err := xml.MarshalIndent(config, "", "    ")
		if (err != nil) {
			panic(err)
			return
		}
		err = ioutil.WriteFile("config.xml", xml, 0666)
		if (err != nil) {
			panic(err)
			return
		}
	} else {
		//Read config file if does exist
		data, err := ioutil.ReadFile("config.xml")
		if (err != nil) {
			panic(err)
			return
		}
		if (string(data) == "") {
			return
		}
		err = xml.Unmarshal(data, &config)
		if (err != nil) {
			panic(err)
			return
		}
	}
	//Override config with flags if set
	if (*portFlag != 0) {
		config.Port = *portFlag
	}
	if (*slic3rPathFlag != "") {
		config.Slic3rPath = *slic3rPathFlag
	}
	//Start HTTP server
	http.HandleFunc("/slice", sliceHandler)
	http.Handle("/gcode/", http.StripPrefix("/gcode/", http.FileServer(http.Dir("gcode"))))
	http.Handle("/stl/", http.StripPrefix("/stl/", http.FileServer(http.Dir("stl"))))
	http.HandleFunc("/gcode", fileListHandler)
	http.HandleFunc("/stl", fileListHandler)
	log.Printf("Slic3r Server binding to port: %d\n", config.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}



func fileListHandler(writer http.ResponseWriter, request *http.Request) {
	if(request.Method == "GET") {
		files, err := ioutil.ReadDir("." + request.URL.String())
		if (err != nil) {
			http.Error(writer, err.Error(), 500)
			return
		}
		var fileList []string
		for _, file := range files {
			fileList = append(fileList, file.Name())
		}
		data, err := json.MarshalIndent(fileList, "", "    ")
		writer.Write(data)
	} else if (request.Method == "DELETE") {

	}
}

func sliceHandler(writer http.ResponseWriter, request *http.Request) {
	//Reject request if it is not a POST request
	if (request.Method != "POST") {
		http.Error(writer, "Request is not a POST request", 400)
		return
	}
	//Get form data
	request.ParseMultipartForm(32 << 20)
	var otherArgs, callbackType, callbackURL string
	var wait bool
	for key, value := range request.Form {
		if (key == "callback" && len(value) > 0) {
			tmp := strings.Split(value[0], ",")
			callbackType = tmp[0]
			callbackURL = tmp[1]
		} else if (key == "wait" && len(value) > 0) {
			if (value[0] == "true") {
				wait = true
			}
			if (value[0] != "true" && value[0] != "false") {
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

