rm -f newoid
rm -f file
dd if=/dev/urandom of=file bs=16M count=1
curl -XPUT --data-binary @file http://127.0.0.1:3000/video/newoid
start=0
end=10485759 # 10M - 1

echo "===check all data==="
curl http://127.0.0.1:3000/calcmd5/video/newoid
md5sum file

sleep 10
echo "===check first 10M==="
curl -H"Range:bytes 0-$end" http://127.0.0.1:3000/calcmd5/video/newoid
dd if=file bs=1M count=10| md5sum -

sleep 10
echo "===check last 6M==="
curl -H"Range:bytes 10485760-16777215" http://127.0.0.1:3000/calcmd5/video/newoid
dd if=file bs=1M count=6 skip=10|md5sum -

