. ./common.sh

rm -f newoid
rm -f file


auth=$(gen_hmac /video/newoid)

dd if=/dev/urandom of=file bs=80M count=1
curl -v -H"Authorization: $auth" -XPUT -i --data-binary @file http://127.0.0.1:3000/video/newoid

wget http://127.0.0.1:3000/video/newoid
md5sum file
md5sum newoid

auth=$(gen_hmac /video/newoid)
curl -H"Authorization:$auth" -XDELETE http://127.0.0.1:3000/video/newoid

