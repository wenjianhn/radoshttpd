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

