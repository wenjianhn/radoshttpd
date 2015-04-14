import hashlib
import hmac
import base64
import sys

message = bytes(sys.argv[1]).encode('utf-8')
secret = bytes(sys.argv[2]).encode('utf-8')

signature = base64.b64encode(hmac.new(secret, message, digestmod=hashlib.sha1).digest())
print(signature)
