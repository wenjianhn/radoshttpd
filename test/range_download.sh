for i in {1..8}
do
    # TODO(wenjianhn): upload a cacheable file
    rm -f newoid2
    axel -q -n $i http://localhost:3000/video/newoid2
    sha1sum newoid2
    rm -f newoid2
done
