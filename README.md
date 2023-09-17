# Stay Focused
> _Keep your webcam focused_

## What?
This app/service monitors for your webcam to be in use and when found to be in use it'll run a command 
to tell your camera to refocus on a given interval. 

## Why?
I use a Logitech webcam with Ubuntu and while it works quite well, it can lose focus if I move around much and 
fail to refocus on its own. Using the `v4l2-ctl` utility the camera can be told to refocus. I've been using that
manually during meetings when my camera gets unfocused and finally decided to write a systemd service to do
that automatically for me. 

## How it works
`stay-focused` will either monitor for a specific process to be running or a specific module to be in use. The 
default is to look for the module `uvcvideo` to be in use. When either the given process or module is detected 
to be in use, `stay-focused` will run the given command to tell your camera to refocus. The interval in which 
checks for camera use is configurable as well as how frequently the camera is told to refocus while the camera
is in use. 

## Defaults:
 - Camera in use checks: Every 1 minute
 - Refocus calls while camera is in use: Every 10 seconds
 - Camera device: /dev/video0
 - Module to check for use: `uvcvideo`

