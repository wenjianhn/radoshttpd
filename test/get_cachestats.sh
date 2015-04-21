echo "Testing GET cache stats"
curl -s -I http://localhost:3000/cachestats | grep "200 OK"
if [ $? -ne 0 ]; then
    echo "Failed. Expected 200 OK"
fi
