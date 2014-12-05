#!/bin/bash
# See how we were called.

. /etc/rc.d/init.d/functions
pidfile=/var/run/wuzei/wuzei.pid
prog=wuzei
wuzei=/usr/bin/wuzei
lockfile=/var/lock/subsys/wuzei
RETVAL=0

start() {
        echo -n $"Starting $prog: "
	daemon --check $prog "nohup $wuzei $OPTIONS < /dev/null > /dev/null 2>&1 &"
	RETVAL=$?
	echo 
        [ $RETVAL = 0 ] && touch ${lockfile}
	return $RETVAL
}

stop() {
	echo -n $"Stopping $prog: "
	killproc -p ${pidfile} -d 10 $httpd
	RETVAL=$?
	echo
	[ $RETVAL = 0 ] && rm -f ${lockfile} ${pidfile}
}

case "$1" in
  start)
	start
	;;
  stop)
	stop
	;;
  status)
        status -p ${pidfile} $wuzei
	RETVAL=$?
	;;
  restart)
	stop
	start
	;;
  *)
	echo $"Usage: $prog {start|stop|restart}"
	RETVAL=2
esac

exit $RETVAL
