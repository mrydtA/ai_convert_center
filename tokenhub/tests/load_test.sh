#!/bin/bash

# TokenHub 压力测试脚本
# 用法：./load_test.sh <API_URL> [OPTIONS]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
API_URL="${1:-http://localhost:8080}"
CONCURRENT_USERS="${2:-50}"
REQUEST_COUNT="${3:-1000}"
TEST_DURATION="${4:-60}"

echo "=========================================="
echo "  TokenHub 压力测试"
echo "=========================================="
echo "API 地址：$API_URL"
echo "并发用户数：$CONCURRENT_USERS"
echo "请求总数：$REQUEST_COUNT"
echo "测试时长：${TEST_DURATION}秒"
echo "=========================================="

# 检查依赖
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}错误：curl 未安装${NC}"
        exit 1
    fi
    
    if ! command -v ab &> /dev/null && ! command -v wrk &> /dev/null; then
        echo -e "${YELLOW}警告：ab 或 wrk 未安装，使用基础 curl 测试${NC}"
        USE_CURL=true
    else
        USE_CURL=false
    fi
}

# 健康检查
health_check() {
    echo -e "\n${GREEN}[1/5] 健康检查${NC}"
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/health" 2>/dev/null || echo "000")
    
    if [ "$response" = "200" ]; then
        echo "✓ 服务健康检查通过"
    else
        echo -e "${RED}✗ 服务健康检查失败 (HTTP $response)${NC}"
        exit 1
    fi
}

# API 端点压测 - 聊天补全
test_chat_completion() {
    echo -e "\n${GREEN}[2/5] 聊天补全接口压测${NC}"
    
    if [ "$USE_CURL" = true ]; then
        # 使用 curl 进行基础压测
        start_time=$(date +%s.%N)
        success_count=0
        fail_count=0
        total_time=0
        
        for i in $(seq 1 $REQUEST_COUNT); do
            request_start=$(date +%s.%N)
            response=$(curl -s -o /dev/null -w "%{http_code}" \
                -X POST "$API_URL/v1/chat/completions" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer test-key" \
                -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}' \
                --connect-timeout 5 \
                --max-time 10 2>/dev/null || echo "000")
            request_end=$(date +%s.%N)
            
            request_time=$(echo "$request_end - $request_start" | bc)
            total_time=$(echo "$total_time + $request_time" | bc)
            
            if [ "$response" = "200" ]; then
                ((success_count++))
            else
                ((fail_count++))
            fi
            
            # 进度显示
            if [ $((i % 100)) -eq 0 ]; then
                echo "  进度：$i/$REQUEST_COUNT (成功:$success_count, 失败:$fail_count)"
            fi
        done
        
        end_time=$(date +%s.%N)
        total_duration=$(echo "$end_time - $start_time" | bc)
        qps=$(echo "scale=2; $REQUEST_COUNT / $total_duration" | bc)
        avg_latency=$(echo "scale=3; $total_time / $REQUEST_COUNT * 1000" | bc)
        
        echo ""
        echo "测试结果:"
        echo "  总请求数：$REQUEST_COUNT"
        echo "  成功数：$success_count"
        echo "  失败数：$fail_count"
        echo "  QPS: $qps"
        echo "  平均延迟：${avg_latency}ms"
        echo "  总耗时：${total_duration}s"
    else
        # 使用 ab 进行压测
        if command -v ab &> /dev/null; then
            ab -n $REQUEST_COUNT -c $CONCURRENT_USERS \
               -H "Authorization: Bearer test-key" \
               -H "Content-Type: application/json" \
               -p <(echo '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}') \
               -T "application/json" \
               "$API_URL/v1/chat/completions" 2>&1 | grep -E "(Requests per second|Time per request|Failed requests)"
        fi
    fi
}

# 并发用户模拟
test_concurrent_users() {
    echo -e "\n${GREEN}[3/5] 并发用户模拟测试${NC}"
    
    # 创建临时文件存储结果
    result_file=$(mktemp)
    
    # 启动多个后台进程模拟并发用户
    for i in $(seq 1 $CONCURRENT_USERS); do
        (
            for j in $(seq 1 10); do
                curl -s -o /dev/null -w "%{http_code},%{time_total}\n" \
                    -X GET "$API_URL/v1/models" \
                    -H "Authorization: Bearer test-key" \
                    --connect-timeout 5 \
                    --max-time 10 2>/dev/null >> "$result_file" || echo "000,0" >> "$result_file"
            done
        ) &
    done
    
    # 等待所有后台进程完成
    wait
    
    # 统计结果
    total_requests=$(wc -l < "$result_file")
    success_requests=$(grep "^200," "$result_file" | wc -l)
    fail_requests=$((total_requests - success_requests))
    
    avg_time=$(awk -F',' '{sum+=$2} END {print sum/NR}' "$result_file")
    
    echo "测试结果:"
    echo "  并发用户数：$CONCURRENT_USERS"
    echo "  总请求数：$total_requests"
    echo "  成功数：$success_requests"
    echo "  失败数：$fail_requests"
    echo "  平均响应时间：$(echo "scale=3; $avg_time * 1000" | bc)ms"
    
    rm -f "$result_file"
}

# 持续负载测试
test_sustained_load() {
    echo -e "\n${GREEN}[4/5] 持续负载测试 (${TEST_DURATION}秒)${NC}"
    
    start_time=$(date +%s)
    request_count=0
    error_count=0
    
    while true; do
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $TEST_DURATION ]; then
            break
        fi
        
        # 每秒发送一批请求
        for i in $(seq 1 10); do
            response=$(curl -s -o /dev/null -w "%{http_code}" \
                -X GET "$API_URL/v1/models" \
                -H "Authorization: Bearer test-key" \
                --connect-timeout 2 \
                --max-time 5 2>/dev/null || echo "000")
            
            ((request_count++))
            if [ "$response" != "200" ]; then
                ((error_count++))
            fi
        done
        
        # 每秒显示进度
        success_rate=$(echo "scale=2; ($request_count - $error_count) * 100 / $request_count" | bc)
        echo -ne "\r进度：${elapsed}s/${TEST_DURATION}s | 请求：$request_count | 错误：$error_count | 成功率：${success_rate}%"
        
        sleep 1
    done
    
    echo ""
    echo ""
    echo "测试结果:"
    echo "  测试时长：${TEST_DURATION}秒"
    echo "  总请求数：$request_count"
    echo "  错误数：$error_count"
    success_rate=$(echo "scale=2; ($request_count - $error_count) * 100 / $request_count" | bc 2>/dev/null || echo "0")
    echo "  成功率：${success_rate}%"
}

# 生成 HTML 报告
generate_report() {
    echo -e "\n${GREEN}[5/5] 生成测试报告${NC}"
    
    report_file="load_test_report_$(date +%Y%m%d_%H%M%S).html"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>TokenHub 压力测试报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 900px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #667eea; text-align: center; }
        .summary { background: #f9f9f9; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .metric { display: inline-block; margin: 10px 20px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #667eea; }
        .metric-label { font-size: 14px; color: #666; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #667eea; color: white; }
        .pass { color: green; }
        .fail { color: red; }
        .timestamp { text-align: center; color: #999; margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 TokenHub 压力测试报告</h1>
        
        <div class="summary">
            <h2>测试概览</h2>
            <div class="metric">
                <div class="metric-value">$API_URL</div>
                <div class="metric-label">API 地址</div>
            </div>
            <div class="metric">
                <div class="metric-value">$CONCURRENT_USERS</div>
                <div class="metric-label">并发用户</div>
            </div>
            <div class="metric">
                <div class="metric-value">$REQUEST_COUNT</div>
                <div class="metric-label">请求总数</div>
            </div>
            <div class="metric">
                <div class="metric-value">${TEST_DURATION}s</div>
                <div class="metric-label">测试时长</div>
            </div>
        </div>
        
        <h2>测试结果详情</h2>
        <table>
            <tr>
                <th>测试项目</th>
                <th>状态</th>
                <th>说明</th>
            </tr>
            <tr>
                <td>健康检查</td>
                <td class="pass">✓ 通过</td>
                <td>服务正常运行</td>
            </tr>
            <tr>
                <td>聊天补全接口</td>
                <td class="pass">✓ 完成</td>
                <td>完成 $REQUEST_COUNT 次请求</td>
            </tr>
            <tr>
                <td>并发用户模拟</td>
                <td class="pass">✓ 完成</td>
                <td>$CONCURRENT_USERS 个并发用户</td>
            </tr>
            <tr>
                <td>持续负载测试</td>
                <td class="pass">✓ 完成</td>
                <td>持续 ${TEST_DURATION}秒负载</td>
            </tr>
        </table>
        
        <div class="timestamp">
            生成时间：$(date '+%Y-%m-%d %H:%M:%S')
        </div>
    </div>
</body>
</html>
EOF

    echo "✓ 报告已生成：$report_file"
    echo -e "${GREEN}==========================================${NC}"
    echo -e "${GREEN}  压力测试完成！${NC}"
    echo -e "${GREEN}==========================================${NC}"
}

# 主函数
main() {
    check_dependencies
    health_check
    test_chat_completion
    test_concurrent_users
    test_sustained_load
    generate_report
}

# 捕获中断信号
trap 'echo -e "\n${YELLOW}测试被用户中断${NC}"; exit 1' INT TERM

main
