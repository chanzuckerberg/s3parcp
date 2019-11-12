# s3parcp

s3parcp is a CLI wrapper around [AWS's Go SDK's Downloader](https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewDownloader). This downloader provides a chunked parallel download implementation from s3 offering speeds faster than [s3cp](https://github.com/aboisvert/s3cp). The API is inspired by `cp`.

Extremely early stage. Don't use this.
