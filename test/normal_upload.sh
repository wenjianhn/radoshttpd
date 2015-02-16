rm -f newoid
rm -f file
dd if=/dev/urandom of=file bs=80M count=1
curl -XPUT -i --data-binary @file http://127.0.0.1:3000/video/newoid
wget http://127.0.0.1:3000/video/newoid
md5sum file
md5sum newoid
curl -XDELETE http://127.0.0.1:3000/video/newoid

