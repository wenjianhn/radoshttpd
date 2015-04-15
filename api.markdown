## Auth Method
	
	All request must have the header Authorization
	Client and server shared the same key
	Use hmac/sha1 and base64 to calculate base64(hmac_sha1(key, urlpath)) = code
	Add a header to http request
	Authorization: code


	PUT /video/newoid HTTP/1.1
	Host: 127.0.0.1:3000
	Authorization: frZ21ljcfDaiSVLh9vutZzBOcEU=
	Content-Length: 83886080
	Content-Type: application/x-www-form-urlencoded
	

	example:
	auth=`python ghmac.py key url.path`
	curl -i -H"Authorivation:$auth" -XDELETE http://127.0.0.1:3000/video/newoid
	

## Put Object

	PUT	/{poolname}/{objectname}
	Option Headers:
	Content-Range: bytes xx-xx/xx
	
	Example:
	#upload the first 100 bytes; total size is 400
	curl -XPUT -H"Content-Range:bytes 0-99/400" --data-binary @file http://wuzei/{poolname}/{objectname}
	#upload the file
	curl -XPUT --data-binary @file http://wuzei/{poolname}/{objectname}

## Get Object

	GET /{poolname}/{objectname}
	Option Headers:
	Range: bytes xx-xx
	
	Example:
	curl -o output http://wuzei/{poolname}/{objectname}
	
## Delete Object
	
	DELETE /{poolname}/{objectname}
	
	Example: 
	curl -XDELETE http://wuzei/{poolname}/{objectname}
	

## Get Object's md5sum by calculating it
	
	GET /calcmd5/{poolname}/{objectname}
	Option Headers:
	Range: bytes xx-xx
	
	Example:
	curl -H"Range:bytes 0-1000" http://127.0.0.1:3000/calcmd5/video/newoid
	curl http://127.0.0.1:3000/calcmd5/video/newoid
	

## Get Object INFO

	GET /info/{poolname}/{objectname}

	RETURN:json
	{"size":xxx}
	

	Example:
	curl http://wuzei/info/{poolname}/{objectname}
	{"size":4096}


## Check Wuzei

	GET /whoareyou
	RETURN:I AM WUZEI
	
	Example:
	curl http://wuzei/whoareyou


## Get Wuzei blocksize

	GET /blocksize
	RETURN:  xxx

	Example:
	curl http://wuzei/blocksize
	
	Usage:
	When using normal upload, but it failed at some point, how could I continue?
	file size = 1000
	curl -XPUT --data-binary @file http://wuzei/{poolname}/{objectname}
	=== FAILED AT SOME POINT ===
	
	#get blocksize and current size
	
	curl -XGET http://wuzei/info/{poolname}/{objectname}
	return REMOTESIZE
	curl -XGET http://wuzei/blocksize
	return BLOCKSIZE

	=== CONTINUE TO UPLOAD ===

	#send data from REMOTESIZE-BLOCKSIZE of file	
	curl -XPUT -H"Content-Range: bytes ($REMOTESIZE-$BLOCKSIZE) - ($FILESIZE) - 1)" --data-binary @file[REMOTESIZE-BLOCKSIZE:]  http://wuzei/{poolname}/{objectname}

