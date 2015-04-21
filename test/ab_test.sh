cd "$(git rev-parse --show-toplevel)"/wuzei

if [ -f /var/run/wuzei/wuzei.pid ]; then
    kill $(cat /var/run/wuzei/wuzei.pid)
    sleep 3
fi

# TODO(wenjianhn): checkout a tag instead of a commit
git checkout 5d5114f1346ad42d582612ff4b2af8ef8750aa0f
rm -f ./wuzei
go build
./wuzei &

rm -f /tmp/expected-sha1sum.txt
filenames=$(striprados -pvideo -l | cut -f1 -d'|')
for file in ${filenames}
do
    hash=$(curl http://localhost:3000/video/${file} | sha1sum )
    echo ${hash}  ${file} >> /tmp/expected-sha1sum.txt
    rm -f ${file}
done

# ==============
if [ -f /var/run/wuzei/wuzei.pid ]; then
    kill $(cat /var/run/wuzei/wuzei.pid)
    sleep 3
fi

# TODO(wenjianhn): checkout a tag instead of a commit
git checkout bc6f4edbcf84ae0d0259cf89ce5619d0143708ae
rm -f ./wuzei
go build
./wuzei &

rm -f /tmp/actual-sha1sum.txt
filenames=$(striprados -pvideo -l | cut -f1 -d'|')
for file in ${filenames}
do
    hash=$(curl http://localhost:3000/video/${file} | sha1sum )
    echo ${hash}  ${file} >> /tmp/actual-sha1sum.txt
    rm -f ${file}
done

diff -u /tmp/expected-sha1sum.txt /tmp/actual-sha1sum.txt

cd -
