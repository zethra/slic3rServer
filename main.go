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
)

type Config struct {
	Port       int
	Slic3rPath string
}

var config = Config{}

func main() {
	//Generate Directories
	if _, err := os.Stat("stl"); os.IsNotExist(err) {
		log.Println("Making STL Directory")
		os.Mkdir("stl", 0777)
	}
	if _, err := os.Stat("gcode"); os.IsNotExist(err) {
		log.Println("Making Gcode Directory")
		os.Mkdir("gcode", 0777)
	}
	//Create config file if does not exist
	if _, err := os.Stat("config.xml"); os.IsNotExist(err) {
		log.Println("Making config")
		config.Port = 8080
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
	//Start HTTP server
	http.HandleFunc("/slice", sliceHandler)
	http.Handle("/gcode/", http.StripPrefix("/gcode/", http.FileServer(http.Dir("gcode"))))
	http.Handle("/stl/", http.StripPrefix("/stl/", http.FileServer(http.Dir("stl"))))
	http.HandleFunc("/gcode", FileListHandler)
	http.HandleFunc("/stl", FileListHandler)
	log.Printf("HTTP server starting on port :%d\n", config.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}

func FileListHandler(writer http.ResponseWriter, request *http.Request) {
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
}

func sliceHandler(writer http.ResponseWriter, request *http.Request) {
	//Reject request if it is not a POST request
	if (request.Method != "POST") {
		http.Error(writer, "Request is not a POST request", 400)
		return
	}
	//Get slic3r args
	request.ParseMultipartForm(32 << 20)
	var otherArgs, callback string
	var wait bool
	for key, value := range request.Form {
		if (key == "callback" && len(value) > 0) {
			callback = value[0]
		} else if (key == "wait" && len(value) > 0 && value[0] == "true") {
			wait = true
		} else {
			if (len(value) > 0) {
				otherArgs += fmt.Sprintf(" --%s %s", key, value[0])
			} else {
				otherArgs += fmt.Sprintf(" --%s", key)
			}
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
	log.Println(header.Header)
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
	if (wait) {
		wg.Wait()
	}
	//Return location of gcode file
	writer.Write([]byte("/gcode/" + fileName + ".gcode"))
	go func() {
		if (callback != "") {
			wg.Wait()
			req, err := http.NewRequest("POST", callback, bytes.NewBuffer([]byte("/gcode/" + fileName + ".gcode")))
			if (err != nil) {
				log.Println(err)
			}
			client := &http.Client{}
			_, err = client.Do(req)
			if (err != nil) {
				log.Println(err)
			}
		}
	}()
}

func exe_cmd(cmd string, wg *sync.WaitGroup) {
	log.Println("command is: ", cmd)
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		log.Printf("%s\n", err)
	}
	log.Printf("%s\n", out)
	wg.Done()
}

