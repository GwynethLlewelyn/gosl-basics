# gosl-basics

A basic example of how to develop external web services for Second Life/OpenSimulator using the Go programming language.

## Installation overview

Requirements:
- You need to have [Go](https://golang.org) installed and configured properly
- Your computer/server needs to have a publicly accessible IP (and set it up with a **dynamic DNS provider** such as [no-ip.com](https://www.noip.com/remote-access)); you _can_ run it from home if you wish

Taking that into account, all you need now to do is `go get git.gwynethllewelyn.net/GwynethLlewelyn/gosl-basics.git` and you _ought_ to have a binary executable file in `~/go/bin` called `gosl-basics`. Just run it!

## Configuration
You can run the executable either as:

  -port string
        Server port (default "3000")
  -server
        Run as server on port 3000
  -shell
        Run as an interactive shell