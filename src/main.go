package main

import (
    "os"
    "log"
    "net/http"
    "io"
)

func main() {
    //    slic3rPath := "~/Downloads/Slic3r/bin/slic3r"

    //Generate Directories
    if _, err := os.Stat("stl"); os.IsNotExist(err) {
        log.Println("Makeing Images Directory")
        os.MkdirAll("stl", 666)
    }
    if _, err := os.Stat("gcode"); os.IsNotExist(err) {
        log.Println("Makeing Scad Directory")
        os.MkdirAll("gcode", 666)
    }


    http.HandleFunc("/", handler)
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
    file, err := os.OpenFile("./stl/" + header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        log.Println(err)
        http.Error(writer, http.StatusText(500), 500)
        return
    }
    defer file.Close()
    io.Copy(file, tmpFile)
}

