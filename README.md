# s3parcp

s3parcp is a CLI wrapper around [AWS's Go SDK's Downloader](https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewDownloader). This downloader provides a chunked parallel download implementation from s3 offering speeds faster than [s3cp](https://github.com/aboisvert/s3cp). The API is inspired by `cp`.

Extremely early stage. Don't use this.

```plain
Usage:
  s3parcp [OPTIONS] [Source] [Destination]

Application Options:
  -p, --part-size=   Part size of parts to be downloaded
  -c, --concurrency= Download concurrency
  -b, --buffer-size= Size of download buffer
      --checksum     Should compare checksum when downloading or place checksum in metadata while uploading

Help Options:
  -h, --help         Show this help message

Arguments:
  Source:            Source to copy from
  Destination:       Destination to copy to (Optional, defaults to source's base name)
```
