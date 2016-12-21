# ccevent
1. 本项目是golang项目，本地编译需要有相应的golang环境
2. 项目依赖fabric，如果本地编译需要有相应的fabric源码
3. 项目需要依赖rabbitmq，其预生产、测试配置文件在./src/amqp.yml，生产环境需要更改此文件配置，放在startup.sh脚本中指定的位置
4. fabric源码的版本基于docker镜像kavlez/fabric-peer:0.0.8
5. 本地编译只需要运行“make clean && make”命令即可。 启动脚本见start_lsn.sh
6. 线上发布、在窗口中启动运行startup.sh 即可。 需要有docker环境
