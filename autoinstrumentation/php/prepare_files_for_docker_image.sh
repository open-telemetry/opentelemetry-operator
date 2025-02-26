#!/usr/bin/env bash
set -xe -o pipefail

PHP_versions=(8.1 8.2 8.3)
libc_variants=(glibc musl)

show_help() {
    echo "Usage: $0 --ext-ver <opentelemetry extension version> --dest-dir <destination directory>"
    echo
    echo "Arguments:"
    echo "    <opentelemetry extension version> - opentelemetry PHP extension version to use. This argument is mandatory."
    echo "    <destination directory> - Directory to store files for docker image. All existing files in this directory will be deleted. This argument is mandatory."
    echo
    echo "Example:"
    echo "  $0 ./files_for_docker_image"
}

parse_args() {
    while [[ "$#" -gt 0 ]]; do
        case $1 in
            --ext-ver)
                opentelemetry_extension_version="$2"
                shift
                ;;
            --dest-dir)
                destination_directory="$2"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown parameter passed: $1"
                show_help
                exit 1
                ;;
        esac
        shift
    done

    if [ -z "${opentelemetry_extension_version}" ] ; then
        echo "<opentelemetry extension version> argument is missing"
        show_help
        exit 1
    fi
    if [ -z "${destination_directory}" ] ; then
        echo "<destination directory> argument is missing"
        show_help
        exit 1
    fi
}

ensure_dir_exists_and_empty() {
    local dir_to_clean="${1:?}"

    if [ -d "${dir_to_clean}" ]; then
        rm -rf "${dir_to_clean}"
        if [ -d "${dir_to_clean}" ]; then
            echo "Directory ${dir_to_clean} still exists. Directory content:"
            ls -l "${dir_to_clean}"
            exit 1
        fi
    else
        mkdir -p "${dir_to_clean}"        
    fi
}

build_native_binaries_for_PHP_version_libc_variant() {
    local PHP_version="${1:?}"
    local libc_variant="${2:?}"
    local dest_dir_for_current_args
    dest_dir_for_current_args="${destination_directory}/native_binaries/PHP_${PHP_version}_${libc_variant}"

    echo "Building extension binaries for PHP version: ${PHP_version} and libc variant: ${libc_variant} to ${dest_dir_for_current_args} ..."

    ensure_dir_exists_and_empty "${dest_dir_for_current_args}"

    local PHP_docker_image="php:${PHP_version}-cli"
    local install_compiler_command=""
    case "${libc_variant}" in
        glibc)
            ;;
        musl)
            PHP_docker_image="${PHP_docker_image}-alpine"
            install_compiler_command="&& apk update && apk add autoconf build-base"
            ;;
        *)
            echo "Unexpected libc variant: ${libc_variant}"
            exit 1
            ;;
    esac

    local current_user_id
    current_user_id="$(id -u)"
    local current_user_group_id
    current_user_group_id="$(id -g)"
    docker run --rm \
        -v "${dest_dir_for_current_args}:/dest_dir" \
        "${PHP_docker_image}" sh -c "\
        mkdir -p /app && cd /app \
        ${install_compiler_command} \
        && pecl install opentelemetry-${opentelemetry_extension_version} \
        && cp /usr/local/lib/php/extensions/no-debug-non-zts-*/opentelemetry.so /dest_dir/ \
        && chown -R ${current_user_id}:${current_user_group_id} /dest_dir/"

    echo "Built extension binaries for PHP version: ${PHP_version} and libc variant: ${libc_variant}"
}

build_native_binaries() {
    echo "Building extension binaries..."

    for PHP_version in "${PHP_versions[@]}" ; do
        for libc_variant in "${libc_variants[@]}" ; do
            build_native_binaries_for_PHP_version_libc_variant "${PHP_version}" "${libc_variant}"
        done
    done

    echo "Built extension binaries"
}

is_earlier_major_minor_version() {
    local lhs_version="${1:?}"
    local rhs_version="${2:?}"
    local lhs_version_major
    lhs_version_major=$(echo "${lhs_version}" | cut -d. -f1)
    local rhs_version_major
    rhs_version_major=$(echo "${rhs_version}" | cut -d. -f1)

    if [ "${lhs_version_major}" -lt "${rhs_version_major}" ]; then
        echo "true"
        return
    fi

    if [ "${lhs_version_major}" -gt "${rhs_version_major}" ]; then
        echo "false"
        return
    fi

    local lhs_version_minor
    lhs_version_minor=$(echo "${lhs_version}" | cut -d. -f2)
    local rhs_version_minor
    rhs_version_minor=$(echo "${rhs_version}" | cut -d. -f2)

    if [ "${lhs_version_minor}" -lt "${rhs_version_minor}" ]; then
        echo "true"
        return
    fi

    echo "false"
}

select_composer_json_for_PHP_version() {
    local PHP_version="${1:?}"
    #
    # Supported instrumentations are different for PHP prior to 8.2 and for PHP 8.2 and later
    # because PHP 8.2 added ability to instrument internal functions
    #
    local is_PHP_version_before_8_2
    is_PHP_version_before_8_2=$(is_earlier_major_minor_version "${PHP_version}" "8.2")
    if [ "${is_PHP_version_before_8_2}" == "true" ]; then
        echo "composer_for_PHP_before_8.2.json"
    else
        echo "composer_for_PHP_8.2_and_later.json"
    fi
}

download_PHP_packages_for_PHP_version() {
    local PHP_version="${1:?}"
    local dest_dir_for_current_args
    dest_dir_for_current_args="${destination_directory}/PHP_packages/PHP_${PHP_version}"

    echo "Downloading PHP packages for PHP version: ${PHP_version} to ${dest_dir_for_current_args} ..."

    ensure_dir_exists_and_empty "${dest_dir_for_current_args}"
    local composer_json_file_name
    composer_json_file_name=$(select_composer_json_for_PHP_version "${PHP_version}")
    local current_user_id
    current_user_id="$(id -u)"
    local current_user_group_id
    current_user_group_id="$(id -g)"
    docker run --rm \
        -v "${dest_dir_for_current_args}:/app/vendor" \
        -v "${PWD}/${composer_json_file_name}:/app/composer.json" \
        -w /app \
        "php:${PHP_version}"-cli sh -c "\
        apt-get update && apt-get install -y unzip \
        && curl -sS https://getcomposer.org/installer | php -- --filename=composer --install-dir=/usr/local/bin \
        && composer --ignore-platform-req=ext-opentelemetry --no-dev install \
        && chown -R ${current_user_id}:${current_user_group_id} ./vendor/"

    echo "Downloaded PHP packages for PHP version: ${PHP_version} to ${dest_dir_for_current_args}"
}

download_PHP_packages() {
    echo "Downloading PHP packages..."

    for PHP_version in "${PHP_versions[@]}" ; do
        download_PHP_packages_for_PHP_version "${PHP_version}"
    done

    echo "Downloaded PHP packages"
}

main() {
    parse_args "$@"

    echo "Preparing files for docker image into directory ${destination_directory} ..."

    ensure_dir_exists_and_empty "${destination_directory}"
    build_native_binaries
    download_PHP_packages

    echo "Prepared files for docker image into directory ${destination_directory}"
}

main "$@"
