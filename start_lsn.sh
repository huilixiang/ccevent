cd ./bin
./ccevent -amqp-file ./amqp.yml \
	-events-address 10.69.42.71:7053 \
	-events-from-chaincode couponcc \
	-listen-to-rejections true \
	-log-file ../logs/event.log \
	-log-level DEBUG
