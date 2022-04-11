# ZBI Kubernetes Klient
To communicate with the Kubernetes API server, the platform uses the client-go package, 
an official Kubernetes client SDK by the Kubernetes community. The package provides 
functionality that can be used to programmatically interact with a Kubernetes cluster. 
Once the Kubernetes manifest files are generated from the templates, this component is 
used to manage their creation, update, retrieval, and update through the API server. 
Afterward, a listener is registered with the API server to monitor the success or failure 
of the resources.

### References
- https://medium.com/@alphegasolutions/introducing-zbi-e9b2eb346a74
- https://medium.com/@alphegasolutions/zbi-architecture-a8dc4aaf4357

## K8s Client

## K8s Informer

## ZBI Klient