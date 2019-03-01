# csi-intel-rsd
A Container Storage Interface ([CSI](https://github.com/container-storage-interface/spec)) Driver for [Intel® Rack Scale Design](https://www.intel.com/content/www/us/en/architecture-and-technology/rack-scale-design-overview.html)(Intel® RSD).

# Development

Requirements:

* Go >= `v1.12` because dependencies are managed with [Go modules](https://github.com/golang/go/wiki/Modules)

Build and verify:

```
$ make all
```

Run:
```
$ ./csirsd -baseurl http://localhost:2443 -username <username> -password <password> --endpoint unix:///tmp/csirsd.sock
2019/01/23 14:16:01 driver.go:121: server started serving on unix:///tmp/csirsd.sock
```

Test CSI API endpoints using [csc utility](https://github.com/rexray/gocsi/tree/master/csc):
```
$ csc identity -e unix:///tmp/csirsd.sock plugin-info
"csi.rsd.intel.com" "0.0.1"

$ csc identity -e unix:///tmp/csirsd.sock plugin-capabilities
CONTROLLER_SERVICE

$ csc identity -e unix:///tmp/csirsd.sock probe
true

$ csc controller -e unix:///tmp/csirsd.sock get-capabilities
&{type:CREATE_DELETE_VOLUME }
&{type:LIST_VOLUMES }

# Create 2 volumes
$ csc controller -e unix:///tmp/csirsd.sock create-volume test --cap SINGLE_NODE_WRITER,block --req-bytes 200000
"14" 200000 "name"="test"

$ csc controller -e unix:///tmp/csirsd.sock create-volume test1 --cap SINGLE_NODE_WRITER,block --req-bytes 100000
"15" 100000 "name"="test1"

# List them
$ csc controller -e unix:///tmp/csirsd.sock list-volumes
"14" 200000 "name"="test"
"15" 100000 "name"="test1"

# Delete one of them
$ csc controller -e unix:///tmp/csirsd.sock delete-volume 14
14

# List again
$ csc controller -e unix:///tmp/csirsd.sock list-volumes
"15" 100000 "name"="test1"

```
