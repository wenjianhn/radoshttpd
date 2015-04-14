key=wuzei
gen_hmac(){
	url=$1
	coded=`python ghmac.py $url $key`
	echo $coded
}
