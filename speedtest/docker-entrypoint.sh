#!/bin/sh
while true; do
    echo "请选择模式："
    echo "1.直接测速  2.指定测速节点  3.退出"
    read -r -p "请输入选项： " choice

    case "$choice" in
        1)
            echo "执行当前测速..."
            speedtest
            ;;
        2)
            read -r -p "请输入测速节点地区，英文(china\beijing\shanghai)，直接回车查询当前 IP 周边节点： " location
            url="https://www.speedtest.net/api/js/servers?engine=js"
            if [ -n "$location" ]; then
                url="${url}&search=${location}"
            fi
            echo "查询节点信息：$url"
            result=$(wget -qO- "$url")
            node_list=$(
                printf '%s\n' "$result" |
                    tr '{' '\n' |
                    sed -n 's/.*"name":"\([^"]*\)".*"country":"\([^"]*\)".*"id":"\([0-9][0-9]*\)".*/\3 -- \2 -- \1/p'
            )
            echo "返回的测速节点信息："
            printf '%s\n' "$node_list"

            while true; do
                read -r -p "请输入测速节点ID，或输入 h 返回主菜单： " server_id

                case "$server_id" in
                    h|H)
                        echo "返回主菜单"
                        break
                        ;;
                    ''|*[!0-9]*)
                        echo "无效节点ID，请输入列表中的数字ID，或输入 h 返回主菜单。"
                        continue
                        ;;
                esac

                if ! printf '%s\n' "$node_list" | grep -q "^${server_id} -- "; then
                    echo "节点ID不在当前列表中，请重新选择。"
                    continue
                fi

                echo "开始指定测速节点 $server_id..."
                if speedtest -s "$server_id"; then
                    break
                fi
                echo "节点测速失败，请重新选择，或输入 h 返回主菜单。"
            done
            ;;
        3)
            echo "退出"
            break
            ;;
        *)
            echo "无效选项，请重新选择."
            ;;
    esac
done
