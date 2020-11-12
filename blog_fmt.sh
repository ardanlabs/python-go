#!/bin/bash
# Format for blog, use as pipe
#   blog_fmt.sh < file.go > /tmp/file.go.n

awk '{gsub("\t", "    ", $0); print NR, $0;}'
