Caching omnitruck proxy
=======================

This repo contains a very simple caching proxy for [Omnitruck API](https://docs.chef.io/api_omnitruck.html)

It forwards all requests to an omnitruck backend (default: `https://omnitruck.chef.io`). The response from the backend is parsed and the proxy downloads and caches the referenced package url.
It then forwards the response from the backend to the client and replaces the `url` field with a local version.

Example:

```
> bin/omnitruck-cache
2018/05/11 15:46:17 Using local cache backend
2018/05/11 15:46:17 Listening on :8080
> curl  'http://localhost:8080/chef/metadata?v=13.9.1&p=ubuntu&pv=16.04&m=x86_64'
sha1    9921fef922bcaf885877e46032ef1ec2fcf37faa
sha256  07a399a16e7eac400a7e1bab7502ffeda33470d37618698d0c8822e410316b99
url     http://localhost:8080/packages/files/stable/chef/13.9.1/ubuntu/16.04/chef_13.9.1-1_amd64.deb
version	13.9.1
```
