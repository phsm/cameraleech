# Cameraleech DVR recorder
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
By default Linux stores written data in so-called "dirty pages" for 3 seconds before forcibly committing them on disk. You can tune dirty pages writeback behavior to keep them in RAM little more in order to accumulate writes. There are 2 sysctl parameters you can tune:
- `vm.dirty_background_ratio` - amount of available RAM in percent which can be used for dirty pages storage. Linux default is 10% but if the server is used exclusively for records storage, you can set it to 60% or more.
- `vm.dirty_expire_centisecs` - time limit (in centiseconds, i.e. 1/100 second). Default is 30 seconds, but you can increase it to 5 or 10 minutes. It increases write performace, but if the server stops unexpectedly you'll lose more data.