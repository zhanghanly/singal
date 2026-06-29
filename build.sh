#!/bin/bash

ProjectName="singal"
BuildVersion=$(git rev-parse HEAD)
BuildTime=$(date "+%FT%T%z")
GitBranch=$(git name-rev --name-only HEAD)
GoVersion="go1.25.linux-amd64"

function build_docker_image() {
    sudo docker build -t $1 \
         --build-arg PROJECT_NAME="${ProjectName}" \
         --build-arg BUILD_VERSION="${BuildVersion}" \
         --build-arg BUILD_TIME="${BuildTime}" \
         --build-arg GIT_BRANCH="${GitBranch}" \
         --build-arg GO_VERSION="${GoVersion}" .
}

function package_docker_image {
    s1=`echo $1 | tr ':' '_'`
    package=$s1".tar.gz"

    sudo docker save -o $package $1
}

function main() {
    images_name=""
    if [ $# -eq 0 ];then
        read -p "Please input docker image name, like singal:v3: " images_name
    else 
        images_name=$1
    fi
    
    echo ${images_name}
    build_docker_image ${images_name}
    package_docker_image ${images_name}
}

main $@
