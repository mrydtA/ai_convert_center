#!/bin/bash

# TokenHub 功能测试脚本
# 用法：./functional_tests.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
API_URL="${API_URL:-http://localhost:8080}"
TEST_API_KEY="${TEST_API_KEY:-sk-test123456}"
MYSQL_HOST="${MYSQL_HOST:-localhost}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-YourPassword}"

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 打印测试结果
print_result() {
    local test_name="$1"
    local result="$2"
    local message="$3"
    
    ((TOTAL_TESTS++))
    
    if [ "$result" = "PASS" ]; then
        ((PASSED_TESTS++))
        echo -e "${GREEN}✓${NC} $test_name"
    else
        ((FAILED_TESTS++))
        echo -e "${RED}✗${NC} $test_name - $message"
    fi
}

# 分隔线
print_header() {
    echo -e "\n${BLUE}=========================================="
    echo -e "  $1"
    echo -e "==========================================${NC}"
}

# ==================== API 测试 ====================

test_api_health() {
    print_header "API 健康检查"
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/health" 2>/dev/null || echo "000")
    
    if [ "$response" = "200" ]; then
        print_result "健康检查接口" "PASS"
    else
        print_result "健康检查接口" "FAIL" "HTTP $response"
    fi
    
    # 测试根路径
    response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/" 2>/dev/null || echo "000")
    if [ "$response" = "200" ] || [ "$response" = "302" ]; then
        print_result "根路径访问" "PASS"
    else
        print_result "根路径访问" "FAIL" "HTTP $response"
    fi
}

test_api_models() {
    print_header "模型列表 API"
    
    response=$(curl -s -w "\n%{http_code}" \
        -X GET "$API_URL/v1/models" \
        -H "Authorization: Bearer $TEST_API_KEY" \
        -H "Content-Type: application/json" 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        print_result "获取模型列表" "PASS"
        
        # 检查返回数据格式
        if echo "$body" | grep -q '"data"'; then
            print_result "响应数据格式" "PASS"
        else
            print_result "响应数据格式" "FAIL" "缺少 data 字段"
        fi
    else
        print_result "获取模型列表" "FAIL" "HTTP $http_code"
    fi
}

test_api_chat_completion() {
    print_header "聊天补全 API"
    
    # 测试简单对话
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "$API_URL/v1/chat/completions" \
        -H "Authorization: Bearer $TEST_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "gpt-3.5-turbo",
            "messages": [
                {"role": "user", "content": "Hello, who are you?"}
            ],
            "max_tokens": 50
        }' 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        print_result "聊天补全请求" "PASS"
        
        # 检查响应结构
        if echo "$body" | grep -q '"choices"' && echo "$body" | grep -q '"message"'; then
            print_result "响应结构验证" "PASS"
        else
            print_result "响应结构验证" "FAIL" "响应格式不正确"
        fi
    else
        print_result "聊天补全请求" "FAIL" "HTTP $http_code"
    fi
    
    # 测试流式响应
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "$API_URL/v1/chat/completions" \
        -H "Authorization: Bearer $TEST_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "gpt-3.5-turbo",
            "messages": [{"role": "user", "content": "Hi"}],
            "stream": true
        }' 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ]; then
        print_result "流式响应支持" "PASS"
    else
        print_result "流式响应支持" "FAIL" "HTTP $http_code"
    fi
}

# ==================== 支付流程测试 ====================

test_payment_order_creation() {
    print_header "支付订单创建"
    
    # 模拟创建充值订单
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "$API_URL/api/payment/topup" \
        -H "Content-Type: application/json" \
        -d '{
            "amount": 100,
            "payment_method": "alipay",
            "user_id": "test_user_001"
        }' 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    # 注意：实际环境中需要真实的支付配置，这里只测试接口可达性
    if [ "$http_code" = "200" ] || [ "$http_code" = "400" ] || [ "$http_code" = "401" ]; then
        print_result "订单创建接口" "PASS" "接口可达 (HTTP $http_code)"
    else
        print_result "订单创建接口" "FAIL" "HTTP $http_code"
    fi
}

test_payment_callback_simulation() {
    print_header "支付回调模拟"
    
    # 模拟支付宝回调
    callback_data="out_trade_no=TEST_ORDER_001&trade_status=TRADE_SUCCESS&total_amount=100.00"
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "$API_URL/api/payment/callback/alipay" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$callback_data" 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    # 回调接口应该返回 success 或 fail
    if [ "$http_code" = "200" ]; then
        print_result "支付宝回调接口" "PASS"
    else
        print_result "支付宝回调接口" "FAIL" "HTTP $http_code"
    fi
    
    # 模拟微信回调
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "$API_URL/api/payment/callback/wechat" \
        -H "Content-Type: application/json" \
        -d '{"out_trade_no":"TEST_ORDER_002","trade_state":"SUCCESS"}' 2>/dev/null)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ]; then
        print_result "微信回调接口" "PASS"
    else
        print_result "微信回调接口" "FAIL" "HTTP $http_code"
    fi
}

# ==================== 数据库测试 ====================

test_database_connection() {
    print_header "数据库连接测试"
    
    # 使用 curl 测试数据库健康端点（如果有）
    response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/admin/health/db" 2>/dev/null || echo "000")
    
    if [ "$response" = "200" ]; then
        print_result "数据库连接检查" "PASS"
    else
        # 尝试直接连接 MySQL（如果安装了 mysql 客户端）
        if command -v mysql &> /dev/null; then
            if mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1" > /dev/null 2>&1; then
                print_result "MySQL 直接连接" "PASS"
            else
                print_result "MySQL 直接连接" "FAIL" "无法连接"
            fi
        else
            print_result "数据库连接检查" "SKIP" "无管理端点且未安装 mysql 客户端"
        fi
    fi
}

test_database_tables() {
    print_header "数据库表结构验证"
    
    if command -v mysql &> /dev/null; then
        # 检查关键表是否存在
        tables=("users" "topup_records" "plans" "payment_flows" "api_keys")
        
        for table in "${tables[@]}"; do
            result=$(mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" \
                -D "tokenhub" -e "SHOW TABLES LIKE '$table'" 2>/dev/null | grep -c "$table" || echo "0")
            
            if [ "$result" -gt 0 ]; then
                print_result "表 $table 存在" "PASS"
            else
                print_result "表 $table 存在" "FAIL" "表不存在"
            fi
        done
    else
        print_result "数据库表验证" "SKIP" "未安装 mysql 客户端"
    fi
}

# ==================== 前端页面测试 ====================

test_frontend_pages() {
    print_header "前端页面加载测试"
    
    pages=(
        "/home/index.html:首页"
        "/topup/index.html:充值页面"
        "/plans/index.html:套餐页面"
        "/docs/index.html:文档中心"
        "/help/index.html:帮助中心"
        "/finance/finance.html:财务报表"
    )
    
    for page_info in "${pages[@]}"; do
        IFS=':' read -r page_path page_name <<< "$page_info"
        
        response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL$page_path" 2>/dev/null || echo "000")
        
        if [ "$response" = "200" ]; then
            print_result "$page_name ($page_path)" "PASS"
        else
            print_result "$page_name ($page_path)" "FAIL" "HTTP $response"
        fi
    done
}

test_static_resources() {
    print_header "静态资源加载测试"
    
    resources=(
        "/assets/css/style.css:主样式表"
        "/assets/css/home.css:首页样式"
        "/assets/js/main.js:主脚本"
    )
    
    for resource_info in "${resources[@]}"; do
        IFS=':' read -r resource_path resource_name <<< "$resource_info"
        
        response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL$resource_path" 2>/dev/null || echo "000")
        
        if [ "$response" = "200" ]; then
            print_result "$resource_name" "PASS"
        else
            print_result "$resource_name" "FAIL" "HTTP $response"
        fi
    done
}

# ==================== 性能基础测试 ====================

test_response_time() {
    print_header "响应时间测试"
    
    endpoints=(
        "/v1/health"
        "/v1/models"
        "/home/index.html"
    )
    
    for endpoint in "${endpoints[@]}"; do
        times=()
        for i in {1..5}; do
            time_ms=$(curl -s -o /dev/null -w "%{time_total}" "$API_URL$endpoint" 2>/dev/null || echo "0")
            times+=("$time_ms")
        done
        
        # 计算平均时间
        avg_time=$(printf '%s\n' "${times[@]}" | awk '{sum+=$1} END {print sum/NR}')
        avg_ms=$(echo "scale=0; $avg_time * 1000" | bc)
        
        if [ "$avg_ms" -lt 500 ]; then
            print_result "$endpoint 平均响应时间 (${avg_ms}ms)" "PASS"
        else
            print_result "$endpoint 平均响应时间 (${avg_ms}ms)" "FAIL" ">500ms"
        fi
    done
}

# ==================== 生成测试报告 ====================

generate_test_report() {
    print_header "测试报告"
    
    echo ""
    echo "测试总览:"
    echo "  总测试数：$TOTAL_TESTS"
    echo -e "  ${GREEN}通过：$PASSED_TESTS${NC}"
    echo -e "  ${RED}失败：$FAILED_TESTS${NC}"
    
    if [ $TOTAL_TESTS -gt 0 ]; then
        pass_rate=$(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc)
        echo "  通过率：${pass_rate}%"
    fi
    
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}============================================${NC}"
        echo -e "${GREEN}  ✓ 所有测试通过！系统运行正常${NC}"
        echo -e "${GREEN}============================================${NC}"
        exit 0
    else
        echo -e "${YELLOW}============================================${NC}"
        echo -e "${YELLOW}  ⚠ 部分测试失败，请检查日志${NC}"
        echo -e "${YELLOW}============================================${NC}"
        exit 1
    fi
}

# ==================== 主函数 ====================

main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════╗"
    echo "║     TokenHub 功能测试套件              ║"
    echo "╚════════════════════════════════════════╝"
    echo -e "${NC}"
    echo "API 地址：$API_URL"
    echo "测试开始时间：$(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    
    # 执行所有测试
    test_api_health
    test_api_models
    test_api_chat_completion
    test_payment_order_creation
    test_payment_callback_simulation
    test_database_connection
    test_database_tables
    test_frontend_pages
    test_static_resources
    test_response_time
    
    # 生成报告
    generate_test_report
}

# 捕获中断信号
trap 'echo -e "\n${YELLOW}测试被用户中断${NC}"; exit 1' INT TERM

# 运行主函数
main
