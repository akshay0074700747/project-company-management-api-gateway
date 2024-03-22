# !/bin/bash

set -e  

buckets=(snapshots-storage-bucket resume-storage-bucket tasks-storage-bucket)

minio="/usr/local/bin/minio"

for bucket in "${buckets[@]}"; do
  if ! $minio mc bucket exists $bucket; then
    $minio mc mb $bucket
  fi
done

echo "Buckets created successfully!"

minio server /data