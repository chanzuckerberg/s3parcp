#!/bin/bash -ex

PLATFORM=Linux_x86_64
VERSION=0.1.4-alpha
RELEASES=https://github.com/chanzuckerberg/s3parcp/releases/download
curl -L $RELEASES/v$VERSION/s3parcp_"$VERSION"_$PLATFORM.tar.gz | tar zx

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y parallel awscli

aws s3 ls

echo $HOME
pwd
ls -a
ls -a /home
ls $HOME/.aws

export AWS_REGION=us-west-2

mkdir maxout

# for i in {1..1000}
# do
#     echo abcdefghijklmnopqrstuvwxyz > maxout/$i
# done
# 
# for i in {1..1000} ; do echo $i ; done | parallel ./s3parcp maxout/{1} s3://idseq-emr-test/maxout/

