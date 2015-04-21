echo "Testing HTTP not found"
curl -s -I http://localhost:3000/Not_Found_26a13914-9ff9-466c-875d-c1d3226ae705  | grep "404 Not Found"
if [ $? -ne 0 ]; then
    echo "Failed. Expected 404 Not Found"
fi
