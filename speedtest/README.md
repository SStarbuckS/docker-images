在 Docker 中运行一个名为 "sstarbucks/speedtest" 标签 "latest" 的容器，并执行速度测试。这个容器会使用宿主机的网络模式（--net=host），这意味着容器内的网络与宿主机是相同的，可以访问宿主机的网络接口。"-it" 标志表示要以交互模式运行容器，并且当容器退出时立即删除 (--rm)。

`docker run -it --rm --net=host sstarbucks/speedtest:latest`

Speedtest CLI 下载版本由 `Dockerfile` 中的 `SPEEDTEST_VERSION` 统一控制，修改该值后重新构建镜像即可切换版本。

选择“指定测速节点”后，在地区输入提示处直接回车，脚本会根据当前 IP 查询周边节点；输入英文地区关键词时按地区查询节点。两种查询结果都按“节点 ID -- 国家 -- 地区”显示，后续需要输入列表中的节点 ID 开始测速。输入无效 ID 时会留在节点选择菜单，输入 `h` 或 `H` 返回主菜单。
