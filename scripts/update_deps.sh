#!/bin/bash

# Used to create the godeps links
# This will rewrite all of the files' imports to link to local packages

godep save -r && echo "Done updating files"
