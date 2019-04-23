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
$ csc identity -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock plugin-info
"csi.rsd.intel.com" "0.0.1"

$ csc identity -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock plugin-capabilities
CONTROLLER_SERVICE

$ csc identity -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock probe
true

$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock get-capabilities
&{type:CREATE_DELETE_VOLUME }
&{type:PUBLISH_UNPUBLISH_VOLUME }
&{type:LIST_VOLUMES }

# Create 2 volumes
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock create-volume test --cap SINGLE_NODE_WRITER,block --req-bytes 4194304
"14" 4194304 "name"="test"

$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock create-volume test1 --cap SINGLE_NODE_WRITER,block --req-bytes 4194304
"15" 4194304 "name"="test1"

# List them
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock list-volumes
"14" 4194304 "name"="test"
"15" 4194304 "name"="test1"

# Delete one of them
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock delete-volume 14
14

# List again
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock list-volumes
"15" 4194304 "name"="test1"

# Publish
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock publish 15 --node-id 1 --cap SINGLE_NODE_WRITER,block --timeout 3m
"15"	"csi.rsd.intel.com/volume-name"="test"

# Stage
$ csc node -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock stage --staging-target-path /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-2f798516-6763-11e9-a6c6-5254000daeea/globalmount 15 --cap SINGLE_NODE_WRITER,mount,ext4
15

# Publish on the node
$ csc node -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock publish --staging-target-path /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-2f798516-6763-11e9-a6c6-5254000daeea/globalmount --target-path /var/lib/kubelet/pods/37ace0a9-6763-11e9-a6c6-5254000daeea/volumes/kubernetes.io~csi/pvc-2f798516-6763-11e9-a6c6-5254000daeea/mount --cap SINGLE_NODE_WRITER,mount,ext4 15

# Unpublish on the node
$ csc node -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock unpublish --target-path /var/lib/kubelet/pods/37ace0a9-6763-11e9-a6c6-5254000daeea/volumes/kubernetes.io~csi/pvc-2f798516-6763-11e9-a6c6-5254000daeea/mount 15
15

# Unstage
$ csc node -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock unstage --staging-target-path /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-2f798516-6763-11e9-a6c6-5254000daeea/globalmount 15
15

# Unpublish
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock unpublish 15 --node-id 1 --timeout 3m
15

# Delete volume
$ csc controller -e unix:///var/lib/kubelet/plugins/csi-intel-rsd/csi.sock delete-volume 15
15

```
