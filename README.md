# s3parcp ![Go](https://github.com/chanzuckerberg/s3parcp/workflows/Go/badge.svg) [![codecov](https://codecov.io/gh/chanzuckerberg/s3parcp/branch/main/graph/badge.svg)](https://codecov.io/gh/chanzuckerberg/s3parcp) [![GitHub license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://github.com/chanzuckerberg/idseq-web/blob/master/LICENSE) ![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)

s3parcp is a CLI wrapper around [AWS's Go SDK's Downloader](https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewDownloader). This downloader provides a chunked parallel download implementation from s3 offering speeds faster than [s3cp](https://github.com/aboisvert/s3cp). The API is inspired by `cp`.

This project is still in pre-release. There are some issues and features I would like to add. Use at your own risk. Bug reports are welcome!

## Installation

### Linux

#### Debian (Ubuntu/Mint)

#### Fedora (RHEL/CentOS)

#### Other Linux

```bash
# Update based on your platform
PLATFORM=#Darwin_i386/Darwin_x86_64/Linux_i386/Linux_x86_64
VERSION=0.2.1-alpha
curl -L https://github.com/chanzuckerberg/s3parcp/releases/download/latest/s3parcp_"$VERSION"_$PLATFORM.tar.gz | tar zx
mv s3parcp ~/.local/bin
```
### MacOS

```bash
brew tap chanzuckerberg/tap
brew install s3parcp
```

### Windows

Unfortunately, due to difficulty with mmap compatibility, Windows is not yet supported. If you want to use s3parcp on Windows 10 I recommend [Windows Subsystem for Linux](https://docs.microsoft.com/en-us/windows/wsl/install-win10). Once you have set this up you can install with the instructions for the Linux distribution you chose. There is currently an open [issue](https://github.com/chanzuckerberg/s3parcp/issues/20) to track this. If this feature is important to you please comment or react on the issue, or make a PR for it.

## Usage

```plain
Usage:
  s3parcp [OPTIONS] [Source] [Destination]

Application Options:
  -p, --part-size=   Part size in bytes of parts to be downloaded
  -c, --concurrency= Download concurrency
  -b, --buffer-size= Size of download buffer in bytes
      --checksum     Compare checksum if downloading or place checksum in metadata if uploading
  -d, --duration     Prints the duration of the download
  -m, --mmap         Use mmap for downloads
  -r, --recursive    Copy directories or folders recursively
      --version      Print the current version
      --s3_url=      A custom s3 API url (also available as an environment variable 'S3PARCP_S3_URL', the flag takes precedence)
      --max-retries= Max per chunk retries
      --disable-ssl  Disable SSL
  -v, --verbose      verbose logging

Help Options:
  -h, --help         Show this help message

Arguments:
  Source:            Source to copy from
  Destination:       Destination to copy to (Optional, defaults to source's base name)
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

#### Using mmap

##### RAM Requirements

Ensure you have enough RAM on your system to hold the entire file. One way to do this is to use the `top` command and check the `KiB Mem`. I also recommend installing `htop` and checking the `Mem` if you prefer a friendlier view. Not having enough RAM may result in swap space being used, slowing your entire system.

##### Optional: Use tmpfs possibly with huge pages

tmpfs is an in-memory filesystem. It allows for faster I/O than a file system on a disk but it is ephemeral, meaning it will be lost when the system shuts down. You can read more about tmpfs [here](https://en.wikipedia.org/wiki/Tmpfs) or [here](https://www.jamescoyle.net/how-to/943-create-a-ram-disk-in-linux). tmpfs can also be configured to use huge pages. Using huge pages can make dealing with large chunks of RAM more efficient because it shrinks the size of the page table. You can read more about huge pages [here](https://wiki.debian.org/Hugepages).

Using tmpfs with mmap can increase download speeds and using huge pages can increase download speeds when dealing with large files.

s3parcp was developed to rapidly download large files into RAM. If you want to start an instance, download a large file onto it, perform computations on it, then shut down the instance, this approach may be for you.

To create a tmpfs mount:

```bash
sudo mkdir /mnt/ramdisk
sudo mount -t tmpfs -o size=1g tmpfs /mnt/ramdisk
```

To create a tmpfs mount with huge pages:

```bash
sudo mkdir /mnt/ramdisk
sudo mount -t tmpfs -o size=1g -o huge=always tmpfs /mnt/ramdisk
```

Any files in /mnt/ramdisk will be stored in your tmpfs file system.

##### Download

```bash
s3parcp --mmap s3://my-bucket/my-object my/local/file
```

## Features

### checksum

This tool comes with a parallelized crc32c checksum validator. The AWS SDK does not support checksums for multipart downloads. If you include the `--checksum` flag when uploading a checksum of your file will be computed and stored in the object's metadata in s3 with the key `x-amz-meta-crc32c-checksum`. When downloading, the `--checksum` flag will compute an independent crc32c checksum of the downloaded file and compare it of the checksum in the object's metadata.

### mmap

s3parcp supports downloading files with mmap. This can improve performance particularly when downloading to an in-memory filesystem such as tmpfs. To leverage this use the `--mmap` flag. Note that doing this results in the entire file being mapped to memory so you must ensure you have enough memory to fit the entire file to use this flag.
