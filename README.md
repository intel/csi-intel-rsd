# Container Storage Interface (CSI) Driver for Intel® Rack Scale Design (Intel® RSD) NVMeoF

## About

Container Storage Interface (CSI) Driver for Intel® Rack Scale Design (Intel® RSD) NVMe is a storage driver for container orchestrators like Kubernetes. It makes remote NVMe storage volumes shared by Intel® RSD hardware available as filesystem volumes to container applications.
The driver communicates with Intel® RSD hardware through Redfish/Swordfish protocols to create storage volumes, list them, attach them to the RSD nodes and delete them. Linux NVMe CLI is used to create NVME over Fabric Connections between RSD compute nodes and an RSD storage subsystem.
The  driver follows the CSI specification by listening for API requests and provisioning volumes accordingly.

## Prerequisites

### Build

This project uses Go modules to manage dependencies. It requires version 1.12 + of Go.\
To build the container image an up to date version of Docker (18.03+) is required.

### Run

The CSI Driver for RSD NVMe is designed to run on a Kubernetes 1.13+ installed on a preconfigured RSD set up.\
The RSD set up should include at least one compute node and a pooled storage node with NVMe storage.\
[RSD Getting Started Guide](https://www.intel.com/content/www/us/en/architecture-and-technology/rack-scale-design/software-getting-started-guide-v2-4.html)\
[RSD Storage API Specifications](https://www.intel.com/content/www/us/en/architecture-and-technology/rack-scale-design/storage-services-api-spec-v2-4.html)\
[RSD REference Implementation](https://www.intel.com/content/www/us/en/architecture-and-technology/rack-scale-design/architecture-spec-v2-4.html)\

Support requests for set up and configuration of RSD should be directed to the [RSD Github repository.](https://github.com/intel/intelRSD)

## Setup

### Quick Start

1) Clone source code:\
```git clone https://github.intel.com/kubernetes-rsd/csi-intel-rsd```\
2) Build the driver:\
```cd csi-intel-rsd && make all```\
3) Create secret for RSD username and password:\
```kubectl create secret generic intel-rsd-secret --from-literal=rsd-username='****' --from-literal=rsd-password='******'```\
4) Create label for RSD node id:\
```kubectl label node --overwrite $(hostname | tr '[:upper:]' '[:lower:]') csi.intel.com/rsd-node=<RSD node id>```\
5) Build driver image:\
```make driver-image```\
6) Run deployment script:\
```cd deployments/kubernetes-1.13 && ./deploy```

### Additional options

csi-intel-rsd driver, node-driver-registrar, csi-provisioner and csi-attacher parameters can be configured in deployments/kubernetes-1.13/driver.yaml\

The following flags can be passed to the driver binary

| Name      |Type| Description   |Default|
|-----------|-----|-----------|--------------|
|baseurl |string |Redfish URL|localhost:2443|
|endpoint|string|CSI endpoint|unix:///var/lib/kubelet/plugins/csi-intel-rsd.sock|
|insecure| flag| Allow connections to https RSD without certificate verification|
|nodeid|string|RSD Node ID|
|password|string|RSD password||
|username|string|RSD username|
|timeout|duration|HTTP Timeout|10s
|help|flag|Print out flag options||

## Usage

The driver enables usage of RSD NVMe over Fabric (NVMeoF) pooled storage in a Kubernetes cluster environment by implementing the CSI specification. RSD NVMeoF storage volumes can be used in Kubernetes pods as dynamically provisioned Persistent Volumes.\
Kubernetes pods can then use the Persistent Volumes through PersistentVolumeClaim.

Usage example can be found in the deployments/kubernetes-1.13/example directory as follows

|File Name |Usage |
|------------|----------------|
|deploy-pvc   |shell script to create StorageClass and ParsitentVolumeClaim|
|storageclass.yaml | Kubernetes StorageClass Spec|
|pvc.yaml | ParsitentVolumeClaim spec for Kubernetes |
|undeploy-pvc | shell script to delete StorageClass and ParsitentVolumeClaim|
|deploy-app | shell scrpt to deploy example application pod|
|app.yaml | example application Kubernetes Pod Spec|
|undeploy-app | shell script to delete application pod|

To deploy application user need to first run 'deploy-pvc' script to deploy the persistent volume.

'deploy-app' will then create an application that requests this volume, causing it to be created through the CSI RSD NVMeoF Driver and attached.

## Components

The full CSI driver functionality comes from a collection of four components as per the CSI spec.

|Name|Description|Source|
|---------------------|------|-----------|
|csi-intel-rsd driver| Driver for CSI interaction with RSD Pooled NVMe Storage|This Repo|
|node-driver-registrar|Sidecar for registering CSI driver with Kubelet|[Link](https://github.com/kubernetes-csi/node-driver-registrar)|
|csi-provisioner|Sidecar for dynamic volume provisioning| [Link](https://github.com/kubernetes-csi/external-provisioner)|
|csi-attacher| Sidecar for attaching volumes to nodes| [Link](https://github.com/kubernetes-csi/external-attacher)|

## Communication and Contribution

Report a bug by filing a new issue.
Contribute by opening a pull request.

Reporting a Potential Security Vulnerability: If you have discovered potential security vulnerability in PMEM-CSI, please send an e-mail to secure@intel.com. For issues related to Intel Products, please visit Intel Security Center.
It is important to include the following details:
•The projects and versions affected
•Detailed description of the vulnerability
•Information on known exploits
Vulnerability information is extremely sensitive. Please encrypt all security vulnerability reports using our PGP key.
A member of the Intel Product Security Team will review your e-mail and contact you to collaborate on resolving the issue. For more information on how Intel works to resolve security issues, see: vulnerability handling guidelines.
