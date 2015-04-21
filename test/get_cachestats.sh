echo "Testing GET cache stats"
curl -s -I -H "x-wuzei-security-token:9f449d76-d083-44fa-84d7-000000000000" http://localhost:3000/cachestats | grep "403 Forbidden"
if [ $? -ne 0 ]; then
    echo "Failed. Expected 403 Forbidden"
fi

curl -s -I -H "x-wuzei-security-token:9f449d76-d083-44fa-84d7-805a4432d284" http://localhost:3000/cachestats | grep "200 OK"
if [ $? -ne 0 ]; then
    echo "Failed. Expected 200 OK"
fi
