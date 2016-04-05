# Slic3r Web Interface
This is and RESTful api for [Slic3r](http://slic3r.org)

## Install
 - Install golang
 - Run `go get github.com/zethra/slic3rServer`
 - Run slic3r server binary
 - Make sure the slic3r binary is in your path

## API
### Slice file
To slice a file send a multipart post request to /slice with the stl file included.  The server will then return the url of the gcode file, which can be downloaded.  
#### Other Parameters
 - wait - if set to true server will wait until slic3r is done to respond
 - callback - if set, server will send a post request containing to gcode file url to the url provided
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

http://localhost:8080/callback
----WebKitFormBoundary7MA4YWxkTrZu0gW
</pre>
Resulting command `slic3r stl/test.stl --repair --layer-height 0.2 --output gcode/test.gcode`

### Get a list of file son the server
Send a get request to /stl or /gcode

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
Send a get request to /stl/filename.stl or /gcode/filename.gcode
#### Sample HTTP requests
<pre>
GET /stl/filename.stl HTTP/1.1
Host: localhost:7766
</pre>

<pre>
GET /gcode/filename.gcode HTTP/1.1
Host: localhost:7766
</pre>
