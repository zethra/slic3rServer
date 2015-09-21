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
)

type Config struct {
    Port int
    Slic3rPath string
}

var config = Config{}

func main() {
    //Generate Directories

    if _, err := os.Stat("stl"); os.IsNotExist(err) {
        log.Println("Makeing Images Directory")
        os.Mkdir("stl", 0777)
    }
    if _, err := os.Stat("gcode"); os.IsNotExist(err) {
        log.Println("Makeing Scad Directory")
        os.Mkdir("gcode", 0666)
    }
    if _, err := os.Stat("config.xml"); os.IsNotExist(err) {
        log.Println("Makeing config")
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

    http.HandleFunc("/", handler)
    http.Handle("/gcode/", http.StripPrefix("/gcode/", http.FileServer(http.Dir("gcode"))))
    log.Printf("HTTP server starting on port :%d\n", config.Port)
    http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}

func handler(writer http.ResponseWriter, request *http.Request) {
    if(request.Method == "GET") {
        http.Error(writer, http.StatusText(400), 400)
        return
    }
    request.ParseMultipartForm(32 << 20)
	var otherArgs string
	for key, value := range request.Form {
		if(len(value) > 0) {
			otherArgs += fmt.Sprintf(" --%s %s", key, value[0])
		} else {
			otherArgs += fmt.Sprintf(" --%s", key)
		}
	}
    tmpFile, header, err := request.FormFile("file")
    if err != nil {
        log.Println(err)
        http.Error(writer, http.StatusText(400), 400)
        return
    }
    defer tmpFile.Close()
    log.Println(header.Header)
    fileName := header.Filename[:(len(header.Filename) - 4)]
    file, err := os.OpenFile("stl/" + header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        log.Println(err)
        http.Error(writer, http.StatusText(500), 500)
        return
    }
    io.Copy(file, tmpFile)
    file.Close()
    args := fmt.Sprintf(" stl/%s.stl %s --output gcode/%s.gcode", fileName, otherArgs, fileName)
    wg := new(sync.WaitGroup)
    wg.Add(1)
    go exe_cmd(config.Slic3rPath + args, wg)
    wg.Wait()
    writer.Write([]byte("/gcode/" + fileName + ".gcode"))
}

func exe_cmd(cmd string, wg *sync.WaitGroup) {
    log.Println("command is: ", cmd, "\n")
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

