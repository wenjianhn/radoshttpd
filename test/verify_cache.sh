echo "Verifying Cache"

# TODO(wenjianhn): only download cacheable files
filenames=$(striprados -pvideo -l | cut -f1 -d'|')
for file in ${filenames}
do
    fresh=$(curl -s http://localhost:3000/video/${file} | sha1sum | awk '{print $1}')
    cache=$(curl -s http://localhost:3000/video/${file} | sha1sum | awk '{print $1}')
    if [ ${fresh} != ${cache} ]; then
        echo Hash of ${file} has changed from ${fresh} to ${cache}
    # else
    #     echo Cache of ${file} looks good.
    fi
done
