# s3parcp

s3parcp is a CLI wrapper around [AWS's Go SDK's Downloader](https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewDownloader). This downloader provides a chunked parallel download implementation from s3 offering speeds faster than [s3cp](https://github.com/aboisvert/s3cp). The API is inspired by `cp`.

```plain
Usage:
  s3parcp [OPTIONS] [Source] [Destination]

Application Options:
  -p, --part-size=   Part size of parts to be downloaded
  -c, --concurrency= Download concurrency
  -b, --buffer-size= Size of download buffer
      --checksum     Should compare checksum when downloading or place checksum in metadata while uploading
  -d, --duration     Prints the duration of the download
  -m, --mmap         Use mmap for downloads

Help Options:
  -h, --help         Show this help message

Arguments:
  Source:            Source to copy from
  Destination:       Destination to copy to (Optional, defaults to source's base name)
```

## checksum

This tool comes with a parallelized crc32c checksum validator. The AWS SDK does not support checksums for multipart downloads. If you include the `--checksum` flag when uploading a checksum of your file will be computed and stored in the object's metadata in s3. When downloading, the `--checksum` flag will compute an independent crc32c checksum of the downloaded file and compare it of the checksum in the object's metadata.

## mmap

s3parcp supports downloading files with mmap. This can improve performance particularly when downloading to an in-memory filesystem such as tmpfs. To leverage this use the `--mmap` flag. Note that doing this results in the entire file being mapped to memory so you must ensure you have enough memory to fit the entire file to use this flag.
