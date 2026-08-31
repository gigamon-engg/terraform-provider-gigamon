#! /usr/bin/env bash

#   Copyright (c) 2017-2026 Gigamon, Inc. All rights reserved.
# 
#   Author: Gigamon Terraform Team (gigamon-terraform-team@gigamon.com)
# 
#   This program is free software: you can redistribute it and/or modify
#   it under the terms of the GNU General Public License as published by
#   the Free Software Foundation, version 3 of the License.
# 
#   This program is distributed in the hope that it will be useful,
#   but WITHOUT ANY WARRANTY; without even the implied warranty of
#   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
#   GNU General Public License for more details.
# 
#   You should have received a copy of the GNU General Public License
#   along with this program. If not, see <https://www.gnu.org/licenses/>


# Generates a release from this branch with the given version. Also tags this to the repo
# release [branch ]  <enable_code_coverage>
#    [branch] is the branch we need to use to do the build
#    <enable_code_coverage> optional parameter that enables code coverage in the binary
#                 that is generated. Accepts true/false as the argument. Defaults to false

RELEASE_VERSION_FILE="release_version.txt"
TERRAFORM_VERSION_FILE="terraform_version.txt"

set -xeuo pipefail

function update_versions {
    local release_version terraform_version
    local release_major release_minor release_patch
    local terraform_major terraform_minor terraform_patch
    local new_release_version new_terraform_version

    release_version=$(tr -d '[:space:]' < "$RELEASE_VERSION_FILE")
    terraform_version=$(tr -d '[:space:]' < "$TERRAFORM_VERSION_FILE")

    if ! [[ "$release_version" =~ ^[0-9]+.[0-9]+.[0-9]{2}$ ]]; then
        echo "Invalid release version: $release_version"
        echo "Expected format: M.m.rr, for example 6.14.02"
        exit 1
    fi

    if ! [[ "$terraform_version" =~ ^[0-9]+.[0-9]+.[0-9]+$ ]]; then
        echo "Invalid Terraform version: $terraform_version"
        echo "Expected format: M.m.xxx, for example 6.14.341"
        exit 1
    fi

    IFS='.' read -r release_major release_minor release_patch <<< "$release_version"
    IFS='.' read -r terraform_major terraform_minor terraform_patch <<< "$terraform_version"

    release_patch=$((10#$release_patch + 1))

    if (( release_patch > 99 )); then
        echo "Release number cannot exceed 99: $release_patch"
        echo "Update the major/minor version in release_version.txt."
        exit 1
    fi

    terraform_patch=$((10#$terraform_patch + 1))

    new_release_version=$(printf "%s.%s.%02d" \
        "$release_major" "$release_minor" "$release_patch")

    new_terraform_version=$(printf "%s.%s.%d" \
        "$release_major" "$release_minor" "$terraform_patch")

    echo "Creating release version: $new_release_version"
    echo "Creating Terraform version: $new_terraform_version"

    printf '%s\n' "$new_release_version" > "$RELEASE_VERSION_FILE"
    printf '%s\n' "$new_terraform_version" > "$TERRAFORM_VERSION_FILE"

    git add "$RELEASE_VERSION_FILE" "$TERRAFORM_VERSION_FILE"
    git commit --message \
        "Version updated: release ${new_release_version}, Terraform ${new_terraform_version}"
    git push
}

# Validates the given argument both in syntax and also ensures clean repo for build
function validate_arguments {

    # Vaidate the number of arguments and the syntax
    if [[ $# -eq 0 ]] || [[ $# -gt 2 ]]; then
        set +x # Stop the echoing so that the output looks clean
        echo "Error: Invalid number of arguments"
        echo "Usage: $0 [branch] <enable_code_coverage>"
        echo "   branch - branch to checkout and build. for e.g. main. Mandatory parameter"
        echo "   enable_code_coverage - optional parameter that enables code coverage"
        echo "       Can be set to true/false. Defaults to false, i.e. code coverage disabled" 
        exit 1
    fi

    if [[ $# -eq 2 ]]; then
        if [[ $2 != "true" ]] && [[ $2 != "false" ]]; then
            set +x # Stop echoing to make the outut look clean
            echo "Invalid value: $2 for enable_code_coverage"
            echo "Usage: $0 [branch] <enable_code_coverage>"
            echo "   branch - branch to checkout and build. for e.g. main. Mandatory parameter"
            echo "   enable_code_coverage - optional parameter that enables code coverage"
            echo "    Can be set to true/false. Defaults to false, i.e. code coverage disabled" 
            exit 1
        fi
    fi

    # Before attempting to build, move to the appropriate directory
    script_source="$( cd "$(dirname "${BASH_SOURCE[0]}" )" && pwd)"
    cd $script_source


    # Checkout the requestd branch and make sure the local repo is clean
    if ! git checkout $1 ; then
        echo "checkout of the requested branch $1 failed. See the above error message"
        exit 1
    fi

    # Pull from upstream if configured; otherwise pull directly from origin/branch.
    if git rev-parse --abbrev-ref --symbolic-full-name "@{u}" > /dev/null 2>&1 ; then
        if ! git pull --ff-only ; then
            echo "pull of the requested branch $1 failed. See the above error message"
            exit 1
        fi
    else
        if ! git pull --ff-only origin $1 ; then
            echo "pull of the requested branch $1 failed. See the above error message"
            exit 1
        fi
    fi

    # Make sure that this is a clean local repo.
    out=`git status --short`
    if [[ "$out" != "" ]] ; then
        echo "your git branch $1 in the local repo is not clean. Cannot build"
        exit 1
    fi

    # Bump up the version by 1, and commit that back to the repo

    update_versions

    release_version=$(tr -d '[:space:]' < "$RELEASE_VERSION_FILE")
    terraform_version=$(tr -d '[:space:]' < "$TERRAFORM_VERSION_FILE")
}

# Given the version, os and arch sets up the artifact for this combination
function build_artifact {
    # Build this combination first
    if ! CGO_ENABLED="0" GOOS=$3 GOARCH=$4 go build $5 -ldflags "-X 'main.version=v$2'" .; then
        echo "Unable to build for $3 and $4"
        exit 1
    fi

    # Form the artifact for this version/os/arch
    build/build.py --binary terraform-provider-gigamon --os $3 --arch $4 --version $2 --base_dir $1
}


# This OS and architectures that we are going to be supporting
declare -A build_variants
build_variants["linux"]="amd64 arm64"
build_variants["darwin"]="amd64 arm64"
build_variants["windows"]="amd64"

# Validate the arguments, and also change our working directory to the root of the git repo
# base_name will contain the directory where the repo is present

validate_arguments "$@"
if [[ $# -eq 2 ]] && [[ ${2} == "true" ]]; then
    code_coverage="-cover"
else
    code_coverage=""
fi

script_source="$( cd "$(dirname "${BASH_SOURCE[0]}" )" && pwd)"
base_dir=`dirname $script_source`

release_version=$(tr -d '[:space:]' < "$RELEASE_VERSION_FILE")
terraform_version=$(tr -d '[:space:]' < "$TERRAFORM_VERSION_FILE")

# Loop over the build variants and set up each of these in the artifact
for os in "${!build_variants[@]}"; do
    declare -a arch_list=(${build_variants[$os]})
    for arch in "${arch_list[@]}"; do
        echo "OS: ${os}, arch: ${arch}"
        build_artifact "$base_dir" "$terraform_version" "$os" "$arch" "$code_coverage"
    done
done

# Tag the repo with this version
git tag --annotate "v${release_version}" --message "Release Version ${release_version}"
git push origin "v${release_version}"

