#!/bin/bash

if ! command -v dca &>/dev/null; then
    echo "dca is not installed. Installing..."
    go install -y github.com/bwmarrin/dca/cmd/dca@latest
    echo "dca installed successfully!"
else
    echo "dca is already installed. Skipping..."
fi

if ! command -v yt-dlp &>/dev/null; then
    echo "yt-dlp is not installed. Installing..."
    sudo add-apt-repository -y ppa:tomtomtom/yt-dlp
    sudo apt update
    sudo apt install -y yt-dlp
    echo "yt-dlp installed successfully!"
else
    echo "yt-dlp is already installed. Skipping..."
fi

if ! command -v ffmpeg &>/dev/null; then
    echo "ffmpeg is not installed. Installing..."
    sudo apt update
    sudo apt install -y ffmpeg
    echo "ffmpeg installed successfully!"
else
    echo "ffmpeg is already installed. Skipping..."
fi
