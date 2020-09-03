# s3parcp [![Latest Version](https://img.shields.io/github/release/chanzuckerberg/s3parcp.svg?style=flat?maxAge=86400)](https://github.com/chanzuckerberg/s3parcp/releases) ![Check](https://github.com/chanzuckerberg/s3parcp/workflows/Check/badge.svg) [![codecov](https://codecov.io/gh/chanzuckerberg/s3parcp/branch/main/graph/badge.svg)](https://codecov.io/gh/chanzuckerberg/s3parcp) [![GitHub license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://github.com/chanzuckerberg/idseq-web/blob/master/LICENSE) ![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)

s3parcp is a CLI wrapper around [AWS's Go SDK's Downloader](https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewDownloader). This downloader provides a chunked parallel download implementation from s3 offering speeds faster than [s3cp](https://github.com/aboisvert/s3cp). The API is inspired by `cp`.

This project is still in pre-release. There are some issues and features I would like to add. Use at your own risk. Bug reports are welcome!

## Installation

### Linux

#### Debian (Ubuntu/Mint)

Download and install the `.deb`:

```bash
RELEASES=chanzuckerberg/s3parcp/releases
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=s3parcp_${VERSION}_linux_amd64.deb
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} -o s3parcp.deb
sudo dpkg -i s3parcp.deb
rm s3parcp.deb
```

#### Fedora (RHEL/CentOS)

Download and install the `.rpm`:

```bash
RELEASES=chanzuckerberg/s3parcp/releases
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=s3parcp_${VERSION}_linux_amd64.rpm
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} -o s3parcp.rpm
sudo rpm -i s3parcp.rpm
rm s3parcp.rpm
```

### MacOS

Install via homebrew:

```bash
brew tap chanzuckerberg/tap
brew install s3parcp
```

### Binary

Download the appropriate binary for your platform:

```bash
RELEASES=chanzuckerberg/s3parcp/releases
PLATFORM=#linux,darwin,windows
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=s3parcp_${VERSION}_${PLATFORM}_amd64.tar.gz
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} | tar zx
```

### Windows

Download the appropriate binary for your platform:

```bash
RELEASES=chanzuckerberg/s3parcp/releases
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=s3parcp_${VERSION}_windows_amd64.tar.gz
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} | tar zx
```

## Usage

```plain
Usage:
  s3parcp [OPTIONS] [Source] [Destination]

Application Options:
  -p, --part-size=                  Part size in bytes of parts to be downloaded
  -c, --concurrency=                Download concurrency
  -b, --buffer-size=                Size of download buffer in bytes
      --checksum                    Compare checksum if downloading or place checksum
                                    in metadata if uploading
  -r, --recursive                   Copy directories or folders recursively
      --version                     Print the current version
      --s3_url=                     A custom s3 API url (also available as an environment
                                    variable 'S3PARCP_S3_URL', the flag takes precedence)
      --max-retries=                Max per chunk retries (default: 3)
      --disable-ssl                 Disable SSL
      --disable-cached-credentials  Disable caching AWS credentials
  -v, --verbose                     verbose logging

Help Options:
  -h, --help                        Show this help message

Arguments:
  Source:                           Source to copy from
  Destination:                      Destination to copy to (Optional, defaults to source's base
                                    name)
```

### Examples

#### Uploading

```bash
s3parcp my/local/file s3://my-bucket/my-object
```

#### Downloading

```bash
s3parcp s3://my-bucket/my-object my/local/file
```

#### Tuning Chunk Parameters

**Note**: These example parameters don't necessarily represent good parameters for your system. s3parcp uses sane defaults so it is recommended to use the default parameters unless you have reason to believe your values will work better.

```bash
PART_SIZE=1048576 # 1 MB
BUFFER_SIZE=10485760 # 10 MB
CONCURRENCY=8
s3parcp \
  --part-size $PART_SIZE \
  --concurrency $CONCURRENCY \
  --buffer-size $BUFFER_SIZE \
  my/local/file s3://my-bucket/my-object
```

#### Using CRC32C Checksum

You must upload your file to s3 with s3parcp and the --checksum flag to use this feature for downloads.

Upload your file:

```bash
s3parcp --checksum my/local/file s3://my-bucket/my-object
```

The checksum should be stored in the s3 object's metadata with the key `x-amz-meta-crc32c-checksum`.

Download your file:

```bash
s3parcp --checksum s3://my-bucket/my-object my/new/local/file
```

## Features

### checksum

This tool comes with a parallelized crc32c checksum validator. The AWS SDK does not support checksums for multipart downloads. If you include the `--checksum` flag when uploading a checksum of your file will be computed and stored in the object's metadata in s3 with the key `x-amz-meta-crc32c-checksum`. When downloading, the `--checksum` flag will compute an independent crc32c checksum of the downloaded file and compare it of the checksum in the object's metadata.
