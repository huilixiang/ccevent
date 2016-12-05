cd ./bin
function start_lsn() {
./ccevent -amqp-file ./amqp.yml \
	-events-address 10.69.42.71:7053 \
	-events-from-chaincode couponcc \
	-listen-to-rejections true \
	-log-file ../logs/event.log \
	-log-level DEBUG
return $?
}

function main() {
	while true
	do
		start_lsn
		echo -e "ccevent exited...$?\n"
		if [ $? -ne 0 ]; then
			sleep 1
		fi
	done

}
#main
start_lsn
