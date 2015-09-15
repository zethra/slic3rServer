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
)

var slic3rPath = "/home/zethra/Downloads/Slic3r/bin/slic3r"

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


    http.HandleFunc("/", handler)
    http.Handle("/gcode/", http.StripPrefix("/gcode/", http.FileServer(http.Dir("gcode"))))
    http.ListenAndServe(":8080", nil)
    log.Println("HTTP Server Started")
}

func handler(writer http.ResponseWriter, request *http.Request) {
    if(request.Method == "GET") {
        http.Error(writer, http.StatusText(400), 400)
        return
    }
    request.ParseMultipartForm(32 << 20)
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
    args := fmt.Sprintf(" stl/%s.stl --output gcode/%s.gcode", fileName, fileName)
    wg := new(sync.WaitGroup)
    wg.Add(1)
    go exe_cmd(slic3rPath + args, wg)
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

