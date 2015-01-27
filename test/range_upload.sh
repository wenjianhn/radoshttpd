rm -f newoid

dd if=/dev/urandom of=file1 bs=100 count=1
dd if=/dev/urandom of=file2 bs=100 count=1
dd if=/dev/urandom of=file3 bs=200 count=1
cat file1 file2 file3 > file

curl -XPUT -i -H"Content-Range:bytes 0-99/400" --data-binary @file1 http://127.0.0.1:3000/video/newoid
curl -XPUT -i -H"Content-Range:bytes 100-199/400" --data-binary @file2 http://127.0.0.1:3000/video/newoid
curl -XPUT -i -H"Content-Range:bytes 200-399/400" --data-binary @file3 http://127.0.0.1:3000/video/newoid

wget http://127.0.0.1:3000/video/newoid
md5sum file
md5sum newoid

curl -XDELETE http://127.0.0.1:3000/video/newoid
