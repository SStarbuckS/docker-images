#!/bin/sh

echo "=========================="
echo "      极简网站导航系统      	"
echo "=========================="
echo "启动时间: $(date)"
echo ""

# 确保数据目录存在
mkdir -p /app/data

# 检查数据目录是否为空（首次启动）
if [ -z "$(ls -A /app/data)" ]; then
    echo "检测到数据目录为空，正在初始化默认数据..."
    
    # 将默认数据复制到挂载的数据目录
    if [ -d "/app/default-data" ]; then
        cp -r /app/default-data/* /app/data/
        echo "默认数据初始化完成！"
        echo "已复制以下文件到数据目录："
        ls -la /app/data/
    else
        echo "警告: 未找到默认数据目录 /app/default-data"
    fi
else
    echo "检测到现有数据，跳过初始化"
    echo "当前数据目录内容："
    ls -la /app/data/
fi

echo ""
echo "=========================="
echo ""

# 启动应用
exec /app/navdesk 