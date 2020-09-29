#!/bin/bash

function tagAndPush {
    buildImage=$1 ## the image that has been built already
    baseImageName=$2 ## the host/namespace/image base to use
    version=$3 ## the version
    targetImage="${baseImageName}:${version}"

    echo "Tagging ${buildImage} as ${targetImage}"
    docker tag "${buildImage}" "${targetImage}"

    echo "Pushing ${targetImage}"
    docker push "${targetImage}"

    ## if we are on a release tag, let's extract the version number
    ## the other possible value, currently, is 'master' (or another branch name)
    ## if we are not running in the CI, it fallsback to the `git describe` above
    if [[ ${version} =~ ^[0-9]+\.[0-9]+ ]]; then
        majorMinor=${BASH_REMATCH[0]}

        if [ "${majorMinor}x" != "x" ]; then
            majorMinorImage="${baseImageName}:${majorMinor}"
            echo "Pushing '${majorMinorImage}'"
            docker tag "${targetImage}" "${majorMinorImage}"

            echo "Pushing ${majorMinorImage}"
            docker push "${majorMinorImage}"
        fi
    fi
}

