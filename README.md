# Slic3r Web Interface
This is and RESTful api for [Slic3r](http://slic3r.org)
##API
####Slice file

To slice a file send a multipart post request to / with the stl file included.  The server will then return the url of the gcode file, which can be downloaded.  Note the http server will not respond until slic3r is done, so make sure you client does not time out.  Additional any other form data included will be used as arguments when running slic3r.  

<pre>
POST / HTTP/1.1
Host: example.com
Content-Type: multipart/form-data; boundary=----WebKitFormBoundaryDeC2E3iWbTv1PwMC

------WebKitFormBoundaryDeC2E3iWbTv1PwMC
Content-Disposition: form-data; name="file"; filename="test.stl"
Content-Type: application/octet-stream

STL FILE CONTENT GOES HERE

------WebKitFormBoundaryDeC2E3iWbTv1PwMC
Content-Disposition: form-data; name="SLIC3R ARG NAME"

"SLIC3R ARG VALUE"
------WebKitFormBoundaryDeC2E3iWbTv1PwMC--
</pre>

####Get file list

stl

<pre>
GET /stl HTTP/1.1
Host: example.com
</pre>

gcode

<pre>
GET /gcode HTTP/1.1
Host: example.com
</pre>

####Dowload sliced files

stl

<pre>
GET /stl/filename.stl HTTP/1.1
Host: example.com
</pre>

gcode

<pre>
GET /gcode/filename.gcode HTTP/1.1
Host: example.com
</pre>
