#!/bin/bash
# Display file for blog format, used in pipe
#   cat file.go | blog_fmt.sh

cat -n | sed -e 's/^  //' -e 's/\t/    /'
