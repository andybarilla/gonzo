#!/bin/sh

# gonzo install script
# Inspired by chezmoi's install script (https://get.chezmoi.io)
# and Claude Code's installation approach
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/andybarilla/gonzo/main/install.sh | sh
#
# Options:
#   -b bindir   Installation directory (default: ./bin or /usr/local/bin with sudo)
#   -d          Enable debug logging
#   -t tag      Install specific version tag (default: latest)
#   -h          Show help
#   -V          Show script version

set -e

SCRIPT_VERSION="1.0.0"
GITHUB_OWNER="andybarilla"
GITHUB_REPO="gonzo"
BINARY_NAME="gonzo"
BINDIR="${BINDIR:-}"
TAGARG="latest"
LOG_LEVEL=2

# Color support (disabled if not a terminal or NO_COLOR is set)
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

tmpdir=""

cleanup() {
    if [ -n "${tmpdir}" ] && [ -d "${tmpdir}" ]; then
        rm -rf -- "${tmpdir}"
    fi
}

trap cleanup EXIT
trap 'exit' INT TERM

usage() {
    this="${1}"
    cat <<EOF
${this}: download and install ${BINARY_NAME}

Usage: ${this} [-b bindir] [-d] [-t tag] [-h] [-V]

Options:
    -b bindir   Installation directory (default: ./bin or /usr/local/bin with sudo)
    -d          Enable debug logging
    -t tag      Install specific version tag (default: latest)
    -h          Show this help message
    -V          Show script version

Examples:
    # Install latest version to ./bin
    ${this}

    # Install to /usr/local/bin (may require sudo)
    ${this} -b /usr/local/bin

    # Install specific version
    ${this} -t v1.0.0

    # Install with debug output
    ${this} -d

Environment Variables:
    BINDIR      Override default installation directory
    NO_COLOR    Disable colored output

EOF
    exit 2
}

version() {
    printf '%s install script version %s\n' "${BINARY_NAME}" "${SCRIPT_VERSION}"
    exit 0
}

main() {
    parse_args "${@}"

    # Set default BINDIR if not specified
    if [ -z "${BINDIR}" ]; then
        if [ -w "/usr/local/bin" ]; then
            BINDIR="/usr/local/bin"
        else
            BINDIR="./bin"
        fi
    fi

    tmpdir="$(mktemp -d)"

    GOOS="$(get_goos)"
    GOARCH="$(get_goarch)"
    check_goos_goarch "${GOOS}/${GOARCH}"

    TAG="$(real_tag "${TAGARG}")"
    VERSION="${TAG#v}"

    log_info "found ${BINARY_NAME} version ${VERSION} for ${GOOS}/${GOARCH}"

    # Determine binary suffix
    BINSUFFIX=""
    case "${GOOS}" in
    windows)
        BINSUFFIX=".exe"
        ;;
    esac

    # Construct the binary name from the release
    RELEASE_BINARY="${BINARY_NAME}-${GOOS}-${GOARCH}${BINSUFFIX}"
    BINARY_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${TAG}/${RELEASE_BINARY}"

    log_debug "downloading from ${BINARY_URL}"

    # Download binary
    http_download "${tmpdir}/${RELEASE_BINARY}" "${BINARY_URL}" || {
        log_crit "failed to download ${BINARY_URL}"
        exit 1
    }

    # Download checksums
    CHECKSUMS_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${TAG}/checksums.txt"
    log_debug "downloading checksums from ${CHECKSUMS_URL}"
    http_download "${tmpdir}/checksums.txt" "${CHECKSUMS_URL}" || {
        log_crit "failed to download checksums"
        exit 1
    }

    # Verify checksum
    hash_sha256_verify "${tmpdir}/${RELEASE_BINARY}" "${tmpdir}/checksums.txt"

    # Create installation directory if it doesn't exist
    if [ ! -d "${BINDIR}" ]; then
        log_debug "creating directory ${BINDIR}"
        mkdir -p "${BINDIR}"
    fi

    # Install the binary
    INSTALLED_BINARY="${BINDIR}/${BINARY_NAME}${BINSUFFIX}"
    log_debug "installing to ${INSTALLED_BINARY}"

    cp "${tmpdir}/${RELEASE_BINARY}" "${INSTALLED_BINARY}"
    chmod +x "${INSTALLED_BINARY}"

    log_info "installed ${INSTALLED_BINARY}"

    # Print success message with PATH hint if needed
    case ":${PATH}:" in
    *":${BINDIR}:"*)
        log_info "run '${BINARY_NAME} --help' to get started"
        ;;
    *)
        log_info "add ${BINDIR} to your PATH to run ${BINARY_NAME}"
        log_info "  export PATH=\"${BINDIR}:\${PATH}\""
        ;;
    esac
}

parse_args() {
    while getopts "b:dht:V" arg; do
        case "${arg}" in
        b) BINDIR="${OPTARG}" ;;
        d) LOG_LEVEL=3 ;;
        h) usage "${0}" ;;
        t) TAGARG="${OPTARG}" ;;
        V) version ;;
        *)
            usage "${0}"
            ;;
        esac
    done
}

get_goos() {
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "${os}" in
    cygwin_nt*) goos="windows" ;;
    mingw*) goos="windows" ;;
    msys_nt*) goos="windows" ;;
    *) goos="${os}" ;;
    esac
    printf '%s' "${goos}"
}

get_goarch() {
    arch="$(uname -m)"
    case "${arch}" in
    aarch64) goarch="arm64" ;;
    arm64) goarch="arm64" ;;
    x86_64) goarch="amd64" ;;
    x86) goarch="amd64" ;;
    i386) goarch="amd64" ;;
    i686) goarch="amd64" ;;
    *) goarch="${arch}" ;;
    esac
    printf '%s' "${goarch}"
}

check_goos_goarch() {
    case "${1}" in
    darwin/amd64) return 0 ;;
    darwin/arm64) return 0 ;;
    linux/amd64) return 0 ;;
    linux/arm64) return 0 ;;
    freebsd/amd64) return 0 ;;
    freebsd/arm64) return 0 ;;
    windows/amd64) return 0 ;;
    windows/arm64) return 0 ;;
    *)
        log_crit "unsupported platform: ${1}"
        log_crit "supported platforms: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, freebsd/amd64, freebsd/arm64, windows/amd64, windows/arm64"
        exit 1
        ;;
    esac
}

real_tag() {
    tag="${1}"
    if [ "${tag}" = "latest" ]; then
        log_debug "fetching latest release tag from GitHub"
        release_url="https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest"
        json="$(http_get "${release_url}" "Accept: application/vnd.github.v3+json")"
        if [ -z "${json}" ]; then
            log_crit "failed to fetch latest release from GitHub"
            exit 1
        fi
        # Extract tag_name from JSON response
        real_tag="$(printf '%s\n' "${json}" | tr -s '\n' ' ' | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"//' | sed 's/".*//')"
        if [ -z "${real_tag}" ]; then
            log_crit "failed to parse release tag from GitHub response"
            exit 1
        fi
        log_debug "found latest tag: ${real_tag}"
        printf '%s' "${real_tag}"
    else
        # Use the tag as-is if it's not "latest"
        printf '%s' "${tag}"
    fi
}

http_get() {
    tmpfile="$(mktemp)"
    if http_download "${tmpfile}" "${1}" "${2}"; then
        cat "${tmpfile}"
        rm -f "${tmpfile}"
        return 0
    fi
    rm -f "${tmpfile}"
    return 1
}

http_download() {
    local_file="${1}"
    source_url="${2}"
    header="${3}"

    log_debug "downloading ${source_url}"

    if is_command curl; then
        http_download_curl "${local_file}" "${source_url}" "${header}"
        return $?
    elif is_command wget; then
        http_download_wget "${local_file}" "${source_url}" "${header}"
        return $?
    fi

    log_crit "neither curl nor wget found, unable to download files"
    return 1
}

http_download_curl() {
    local_file="${1}"
    source_url="${2}"
    header="${3}"

    if [ -z "${header}" ]; then
        code="$(curl -w '%{http_code}' -fsSL -o "${local_file}" "${source_url}" 2>/dev/null)" || {
            log_debug "curl failed for ${source_url}"
            return 1
        }
    else
        code="$(curl -w '%{http_code}' -fsSL -H "${header}" -o "${local_file}" "${source_url}" 2>/dev/null)" || {
            log_debug "curl failed for ${source_url}"
            return 1
        }
    fi

    if [ "${code}" != "200" ]; then
        log_debug "received HTTP status ${code} for ${source_url}"
        return 1
    fi
    return 0
}

http_download_wget() {
    local_file="${1}"
    source_url="${2}"
    header="${3}"

    if [ -z "${header}" ]; then
        wget -q -O "${local_file}" "${source_url}" 2>/dev/null || return 1
    else
        wget -q --header "${header}" -O "${local_file}" "${source_url}" 2>/dev/null || return 1
    fi
    return 0
}

hash_sha256() {
    target="${1}"
    if is_command sha256sum; then
        hash="$(sha256sum "${target}")" || return 1
        printf '%s' "${hash}" | cut -d ' ' -f 1
    elif is_command shasum; then
        hash="$(shasum -a 256 "${target}" 2>/dev/null)" || return 1
        printf '%s' "${hash}" | cut -d ' ' -f 1
    elif is_command sha256; then
        hash="$(sha256 -q "${target}" 2>/dev/null)" || return 1
        printf '%s' "${hash}" | cut -d ' ' -f 1
    elif is_command openssl; then
        hash="$(openssl dgst -sha256 "${target}" 2>/dev/null)" || return 1
        # openssl output format: SHA256(file)= hash
        printf '%s' "${hash}" | sed 's/.*= //'
    else
        log_crit "no SHA256 command found (tried sha256sum, shasum, sha256, openssl)"
        return 1
    fi
}

hash_sha256_verify() {
    target="${1}"
    checksums="${2}"
    basename="${target##*/}"

    log_debug "verifying checksum for ${basename}"

    want="$(grep "${basename}" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)"
    if [ -z "${want}" ]; then
        log_crit "checksum not found for ${basename} in checksums file"
        return 1
    fi

    # Validate checksum format (must be 64 hex characters)
    if ! printf '%s' "${want}" | grep -qE '^[a-f0-9]{64}$'; then
        log_crit "invalid checksum format in checksums file"
        log_crit "  got: ${want}"
        return 1
    fi

    got="$(hash_sha256 "${target}")"
    if [ "${want}" != "${got}" ]; then
        log_crit "checksum verification failed for ${basename}"
        log_crit "  expected: ${want}"
        log_crit "  got:      ${got}"
        return 1
    fi

    log_debug "checksum verified for ${basename}"
    return 0
}

is_command() {
    command -v "${1}" >/dev/null 2>&1
}

log_debug() {
    [ 3 -le "${LOG_LEVEL}" ] || return 0
    printf "${BLUE}[debug]${NC} %s\n" "${*}" 1>&2
}

log_info() {
    [ 2 -le "${LOG_LEVEL}" ] || return 0
    printf "${GREEN}[info]${NC} %s\n" "${*}" 1>&2
}

log_err() {
    [ 1 -le "${LOG_LEVEL}" ] || return 0
    printf "${YELLOW}[error]${NC} %s\n" "${*}" 1>&2
}

log_crit() {
    [ 0 -le "${LOG_LEVEL}" ] || return 0
    printf "${RED}[critical]${NC} %s\n" "${*}" 1>&2
}

main "${@}"
