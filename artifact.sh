#!/bin/bash -e

wget -q -O artifact_go.sh 'https://raw.githubusercontent.com/mdblp/tools/feature/add-platform-project/artifact/artifact_go.sh'
chmod +x artifact_go.sh

. ./version.sh
./artifact_go.sh
