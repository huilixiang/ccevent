#!/bin/bash
#ccevent代码版本号
CCEVENT_TAG=1.0.0
#gitlab地址
CCEVENT_REPO=http://gitlab.intra.gomeplus.com/huajie/ccevent.git
#依赖的fabric镜像
DEP_IMG=kavlez/fabric-peer:0.0.8
#启动后的镜像名称
CONTAINER_NAME=ccevent
#fabric vp0的ip地址
VP_HOST=10.69.42.71
#fabric vp的事件发端端口
VP_PORT=7053
#要监听的chaincode name
CHAINCODE_NAME=couponcc
#ccevent依赖的rabbitmq配置
CONF_PATH='/etc/blockchain/ccevent/amqp.yml'
if [ ! -f $CONF_PATH ]; then
	echo $CONF_PATH" is not existed."
	exit 666
fi
#ccevent的日志输出目录
LOG_PATH='/var/blockchain/logs/ccevent/'
if [ ! -d $LOG_PATH ]; then
	echo ''$LOG_PATH" is not existed." 
	exit 233
fi
#启动镜像
docker pull $DEP_IMG 
docker run --name=${CONTAINER_NAME} -itd -v $LOG_PATH:$LOG_PATH -v $CONF_PATH:$CONF_PATH $DEP_IMG
#编译ccevent并启动
docker exec $(docker ps -aqf name=$CONTAINER_NAME) /bin/bash -c "cd /go/src/github.com && git clone ${CCEVENT_REPO} && cd ccevent && git checkout ${CCEVENT_TAG} && make clean && make && cd output && ./ccevent -amqp-file ${CONF_PATH} -events-address ${VP_HOST}:${VP_PORT} -events-from-chaincode ${CHAINCODE_NAME} -listen-to-rejections true -log-file ${LOG_PATH}/ccevent.log -log-level INFO "

