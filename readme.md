# Cameraleech CCTV recorder [![Go Report Card](https://goreportcard.com/badge/github.com/phsm/cameraleech)](https://goreportcard.com/report/github.com/phsm/cameraleech)
Its a simple video recorder leveraging FFMPEG to retrieve video streams from surveillance cameras and store them on disk. 
In a nutshell, its nothing but a smart ffmpeg launcher

## Features
- high-performance: have been tested on 500 cameras on a single server
- open format video files: can be viewed by any media player
- graceful reload: adding or removing cameras on the fly, without restarting.
- metrics for monitoring: zabbix low-level discovery JSON, received frames count, dropped, duplicate frames etc.

The idea behind of cameraleech is simple: read the config with camera names and URLs, launch ffmpeg process per camera, restart if it crashes and collect its statistics.
For convenience, records are stored in segments (1 hour length by default)

Each camera record is stored in "storagePath/_cameraname_/_YYYY-MM-DD_/_segment start time_.mkv"

Tested on Linux. Work on other OSes isn't guaranteed.

## Quickstart
1. Download the archive from releases page, unpack to some directory.
2. Ensure you have ffmpeg installed in your system. I recommend using static ffmpeg build instead. 
3. Edit the configuration file. 
4. Put the cameraleech.service into /etc/systemd/system and edit it, changing paths accordingly.
5. Launch cameraleech service.

## Storage performance
By default Linux stores written data in so-called "dirty pages" for 30 seconds before forcibly committing them on disk. You can tune dirty pages writeback behavior to keep them in RAM little more in order to accumulate writes. There are 2 sysctl parameters you can tune:
- `sysctl -w vm.dirty_ratio=80` - percentage of your RAM which can be left unwritten to disk.
- `sysctl -w vm.dirty_background_ratio=50` - percentage of yout RAM when background writer have to kick in and start writes to disk. Make it way above the value you see in `/proc/meminfo|grep Dirty` so that it doesn't interefere with dirty_expire_centisecs explained below
- `sysctl -w vm.dirty_expire_centisecs=$(( 10*60*100 ))` - allow page to be left dirty no longer than 10 mins. If unwritten page stays longer than time set here, kernel starts writing it out.

_The sysctl parameters descriptions are borrowed [here](https://github.com/lomik/go-carbon#os-tuning)_
