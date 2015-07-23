#!/bin/bash

# Start/stop wuzei daemons
# chkconfig: 2345 60 80

### BEGIN INIT INFO
# Provides:          wuzei
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Required-Start:    $remote_fs $named $network $time
# Required-Stop:     $remote_fs $named $network $time
# Short-Description: Start wuzei daemons at boot time
# Description:       Enable Ceph distributed file system services.
### END INIT INFO


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
