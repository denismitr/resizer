#!/bin/bash
set -e

mongo -- <<EOF
    use resizer;
    var resizer = db.getSiblingDB('resizer');
    resizer.createCollection('images');
EOF