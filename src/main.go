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
    "encoding/json"
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
    http.Handle("/", http.FileServer(http.Dir("web")))
    http.Handle("/stl/", http.StripPrefix("/stl/", http.FileServer(http.Dir("stl"))))
    http.Handle("/gcode/", http.StripPrefix("/gcode/", http.FileServer(http.Dir("gcode"))))
    http.HandleFunc("/api/uploadAndSlice", uploadAndSlice)
    http.HandleFunc("/api/stl/list", getStls)
    http.HandleFunc("/api/gcode/list", getGcodes)
    log.Printf("HTTP server starting on port :%d\n", config.Port)
    http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}

func getStls(writer http.ResponseWriter, request *http.Request) {
    var fileNames []string
    files, err := ioutil.ReadDir("./stl")
    if(err != nil) {
        http.Error(writer, http.StatusText(400), 400)
    }
    for _, f := range files {
            fileNames = append(fileNames, f.Name())
    }
    json, err := json.MarshalIndent(fileNames, "", "    ")
    if(err != nil) {
        http.Error(writer, http.StatusText(400), 400)
    }
    writer.Write(json)
}

func getGcodes(writer http.ResponseWriter, request *http.Request) {
    var fileNames []string
    files, err := ioutil.ReadDir("./gcode")
    if(err != nil) {
        http.Error(writer, http.StatusText(400), 400)
    }
    for _, f := range files {
            fileNames = append(fileNames, f.Name())
    }
    json, err := json.MarshalIndent(fileNames, "", "    ")
    if(err != nil) {
        http.Error(writer, http.StatusText(400), 400)
    }
    writer.Write(json)
}
    

func uploadAndSlice(writer http.ResponseWriter, request *http.Request) {
    //Reject request if it is not a POST request
    if(request.Method != "POST") {
        http.Error(writer, http.StatusText(400), 400)
        return
    }
    //Get slic3r args
    request.ParseMultipartForm(32 << 20)
    var otherArgs string
	for key, value := range request.Form {
		if(len(value) > 0) {
			otherArgs += fmt.Sprintf(" --%s %s", key, value[0])
		} else {
			otherArgs += fmt.Sprintf(" --%s", key)
		}
	}
    //Get STL file
    tmpFile, header, err := request.FormFile("file")
    if err != nil {
        log.Println(err)
        http.Error(writer, http.StatusText(400), 400)
        return
    }
    defer tmpFile.Close()
    log.Println(header.Header)
    fileName := header.Filename[:(len(header.Filename) - 4)]
    //fileName := header.Filename
    file, err := os.OpenFile("stl/" + header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        log.Println(err)
        http.Error(writer, http.StatusText(500), 500)
        return
    }
    io.Copy(file, tmpFile)
    file.Close()
    //Run slic3r with STL file and args
    args := fmt.Sprintf(" \"stl/%s\" %s --output \"gcode/%s.gcode\"", header.Filename, otherArgs, fileName)
    wg := new(sync.WaitGroup)
    wg.Add(1)
    go exe_cmd(config.Slic3rPath + args, wg)
    wg.Wait()
    //Return location of gcode file
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

