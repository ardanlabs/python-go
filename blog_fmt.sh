#!/bin/bash
# Format for blog, use as pipe
#   cat file.go | blog_fmt.sh

awk '{gsub("\t", "    ", $0); print NR, $0;}'
