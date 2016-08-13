# Slic3r Web Interface [![Build Status](https://travis-ci.org/zethra/slic3rServer.svg?branch=master)](https://travis-ci.org/zethra/slic3rServer)
This is and RESTful api for [Slic3r](http://slic3r.org)

## Install
### ARCH
 - An AUR package is avalible here: [AUR](https://aur.archlinux.org/packages/slic3r-server/)

### Manual
 - Install the [go compiler](http://golang.org) and set up the go environment
 - Run `go get github.com/zethra/slic3rServer`
 - Run `slic3rServer` binary

## Configuration
The first time slic3r server runs it generates a config.xml file where various config options can be set

 - port: the port slic3r server will bind to - 7766 be default
 - slic3rPath: the path to the slic3r binary - slic3r (in system path) be default

## Flags
 - All for Slic3r Server's config options con be overridden by setting a flag of the same name to the desired value
 - If the debug flag is set Slic3r Server will output more information about what it's doing

## API
### Slice file
To slice a file send a multipart post request to /slice with the stl file included.  The server will then return the url of the gcode file, which can be downloaded.  
#### Other Parameters
 - wait - if set to true server will wait until slic3r is done to respond
 - callback - this parameter has two parts spereated by a comma, a type and a url.  If the type is url Slic3r Server will send a post request containing to gcode file url to the url provided when slic3r is done. If the type is file Slic3r Server will send a multipart post request containing to gcode file to the url provided when slic3r is done.  Note Slic3r Server will not return an error to the client if callback fails unless wait is set to true
 - Any other parameters set will be used as parameters for slic3r
 
#### Sample HTTP Request
<pre>
POST /slice HTTP/1.1
Host: localhost:7766
Cache-Control: no-cache
Postman-Token: da439a4f-572d-5642-4ebd-80d765450dd8
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW

----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="file"; filename=""
Content-Type: 


----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="repair"


----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="layer-height"

0.2
----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="wait"

true
----WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="callback"

url,http://localhost:8080/callback
----WebKitFormBoundary7MA4YWxkTrZu0gW
</pre>
Resulting command `slic3r stl/test.stl --repair --layer-height 0.2 --output gcode/test.gcode`

#### Response
 - Slic3r Server will respond with the url the goce file will be available at when slic3r is done
 - Slic3r Server may return a 500 error if something goes wrong.  Check the log for specifics

#### Callback
 - The URL callback body will contain the url of the gcode file
 - The file callback will be a multipart request with the file parameter containing the gcode file
 - Both types may have the parameter `error` set to `true` if slicing failed or something went wrong

### Get a list of files on the server
Send a GET request to /stl or /gcode

#### Sample HTTP requests
<pre>
GET /stl HTTP/1.1
Host: localhost:7766
</pre>

<pre>
GET /gcode HTTP/1.1
Host: localhost:7766
</pre>

#### Download files
Send a GET request to /stl/filename.stl or /gcode/filename.gcode
#### Sample HTTP requests
<pre>
GET /stl/filename.stl HTTP/1.1
Host: localhost:7766
</pre>

<pre>
GET /gcode/filename.gcode HTTP/1.1
Host: localhost:7766
</pre>

#### Delete file
Send a DELETE request to the url of the file you want to delete
<pre>
DELETE /stl/test.stl HTTP/1.1
Host: localhost:7766
</pre>

<pre>
DELETE /gcode/test.gcode HTTP/1.1
Host: localhost:7766
</pre>


### Clear all files
Send a delete request to `/stl` or `/gcode` to detele all of those repective files
<pre>
DELETE /stl HTTP/1.1
Host: localhost:7766
</pre>

<pre>
DELETE /gcode HTTP/1.1
Host: localhost:7766
</pre>
