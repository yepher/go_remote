# Go Remote

[Remote.it/](https://remote.it/) is a nice service for getting access to your network without being able to punch a hole in the your firewall.

It is sort of a pain to have to open the web browser to setup the connection. So this is a quickly thrown together GoLang code to setup the connection and output my preferred SSH command with arguments.

## Setup

1. Create the directory `~/.goremote`
2. Copy config `cp config.json ~/.goremote`
3. Edit `config.json` and add your remote.it credential


You can customize the ssh command that will be printed to the screen by adding/removing additional values to the SSH template:

` "SSH_template":"ssh -p ${PORT} pi@${HOST} -L 5900:127.0.0.1:5900  -L 8800:192.168.30.1:80 -L 8801:192.168.30.189:80 -o \"UserKnownHostsFile /dev/null\""`

This make it easy to access VNC on my Raspberry Pi and sets up easy access to two web servers on the network.

## Usage

**List Devices**

Running `goRemote` without any command-line options will list all your devices

**Connect Device**

Use the `-device=<DEVICE_ID>` to connect a device.


## Building

```
# Initial setup
go get github.com/tkanos/gonfig
go get gopkg.in/oleiade/reflections.v1

# Building
go build -o goRemot
```


