# Arxibana

This project is inspired by arxivSanity preserver project by Karpathy. However, there are some key differences:
- the arxiv scraper is written in Go instead of Python
- its not a web app; instead using Kibana dashboard as the UI


## Overview

The go script `arxivProcessing.go` downloads arxiv paper data every 24 hours and stores it in an elasticsearch index, which is used for the Kibana dashboard.

By default, the go script is configured to download 10 latest papers for the categories: Databases(cs.DB) and Distributed computing (cs.DS). At the start, it downloads 500 recent ones and then setiches to 24 hours cadence. 
It won't add paper if that paper is already added.

## How to use it
- Clone the repo
### For macOS
	```
	```
### For raspberry-pi
    ```
    ```
- Run `docker-compose up -d` (Modify Dockerfile if you want to change the categories)

