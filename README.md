# Overview
Application that reads from AMI socket, add timestamp then send the data to message queue (currently RabbitMQ).

# For Developers

## Setup

### Prerequisite
1. Access to [AMI Reader](https://github.com/callcruncher/ami-reader).
2. Install [GIT](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
3. Install [Go 1.13.4](https://golang.org/doc/install) or higher.
4. Set the GOPATH and add go binaries. Below is the sample config in ~/.bash_profile in mac osx.
```bash
export GOROOT=~/go
export GOPATH=~/workspace/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```
5. Install [dep](https://golang.github.io/dep/docs/installation.html). For Windows users, you can download from [releases](https://github.com/golang/dep/releases) page. Please take some time to fully read the usage of [go dep](https://golang.github.io/dep/docs/daily-dep.html),  especially the section on how to add new dependencies.

### Steps

1\. Checkout the [AMI Reader](https://github.com/callcruncher/ami-reader)
```bash 
cd $GOPATH/src
git clone git@github.com/callcruncher/ami-reader.git
```
2\. Pull the project dependencies
```bash
  $ cd ami-reader
  $ dep ensure
``` 

## How to build the app

Just execute `go build .` to generate an executable binary file called `ami-reader`. To target other OS or Arch. See https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

## How to run the app

Make sure the following environment variables are set:  

| Name | Description |
| ---- | ----------- |
| AMI_CONF_PATH | Full path of Asterisk manager conf. Defaults to `/etc/asterisk/manager.conf` |
| AMI_ADMIN_USER | Admin user set in asterisk manager conf section. Defaults to `admin` |
| AMI_HOST | AMI or asterisk host. |
| HOST_DEVICE_ID | Host device id |  

You can set the environment variables in your machine's environment or by creating `config.json` (see `config.json.sample`). Note: Environment variables are resolved from system's environment variable then from `config.json` if it exists.

To run the app, you can execute `go run main.go` or execute the binary file generated from above - `./ami-reader`.

## Notes

App version is managed inside `main.go`. Update appropriately should there be eve**nt related contract updates.  