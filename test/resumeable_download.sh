. ./common.sh

rm -f testrd
rm -f file

auth=$(gen_hmac /video/testrd)

dd if=/dev/urandom of=file bs=64M count=4
curl -H"Authorization: $auth" -XPUT -i --data-binary @file http://127.0.0.1:3000/video/testrd
expected=$(sha1sum file | awk '{print $1}')
rm -f file


# FIXME(wenjianhn): timeout (GNU coreutils) 8.4 doesn't support floating point number,
# so the minimize duration is 1 second

timeout 1s aria2c http://localhost:3000/video/testrd
aria2c http://localhost:3000/video/testrd
actual=$(sha1sum testrd | awk '{print $1}')
rm -f testrd

if [ ${expected} != ${actual} ]; then
    echo Failed to resuming download
else
    echo Test passed
fi

curl -XDELETE -H"Authorization: $auth" http://127.0.0.1:3000/video/testrd
