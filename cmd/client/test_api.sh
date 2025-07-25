#!/bin/bash

# 机器人客户端API测试脚本
# 测试所有HTTP API接口

API_BASE="http://localhost:8080/api/v1"

echo "🤖 机器人客户端API测试"
echo "API地址: $API_BASE"
echo "=================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_api() {
    local method=$1
    local endpoint=$2
    local description=$3
    local data=$4
    
    echo -e "\n${BLUE}测试: $description${NC}"
    echo "请求: $method $API_BASE$endpoint"
    
    if [ "$method" = "POST" ]; then
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data")
        else
            response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE$endpoint")
        fi
    else
        response=$(curl -s -w "\n%{http_code}" "$API_BASE$endpoint")
    fi
    
    # 分离响应体和状态码
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    echo "状态码: $http_code"
    echo "响应:"
    echo "$response_body" | jq '.' 2>/dev/null || echo "$response_body"
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✅ 成功${NC}"
    else
        echo -e "${RED}❌ 失败${NC}"
    fi
}

# 检查服务是否运行
echo "检查API服务状态..."
if curl -s "$API_BASE/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ API服务正在运行${NC}"
else
    echo -e "${RED}❌ API服务未运行，请先启动机器人客户端${NC}"
    echo "启动命令: ./robot_client"
    exit 1
fi

echo -e "\n${YELLOW}开始API测试...${NC}"

# 1. 测试生命值查询
test_api "GET" "/health" "生命值查询"

# 2. 测试弹药查询
test_api "GET" "/ammo" "弹药数量查询"

# 3. 测试射击
test_api "POST" "/shoot" "执行射击"

# 4. 测试更换弹药
test_api "POST" "/ammo/change" "更换弹药"

# 5. 再次查询状态
echo -e "\n${YELLOW}测试后状态查询...${NC}"
test_api "GET" "/health" "射击后生命值查询"
test_api "GET" "/ammo" "射击后弹药查询"

# 6. 测试错误情况
echo -e "\n${YELLOW}测试错误情况...${NC}"

# 测试不支持的方法
echo -e "\n${BLUE}测试: 射击接口使用GET方法${NC}"
response=$(curl -s -w "\n%{http_code}" "$API_BASE/shoot")
http_code=$(echo "$response" | tail -n1)
response_body=$(echo "$response" | head -n -1)
echo "状态码: $http_code"
echo "响应: $response_body"
if [ "$http_code" = "405" ]; then
    echo -e "${GREEN}✅ 正确返回405错误${NC}"
else
    echo -e "${RED}❌ 未正确处理错误${NC}"
fi

# 测试不存在的接口
echo -e "\n${BLUE}测试: 访问不存在的接口${NC}"
response=$(curl -s -w "\n%{http_code}" "$API_BASE/nonexistent")
http_code=$(echo "$response" | tail -n1)
echo "状态码: $http_code"
if [ "$http_code" = "404" ]; then
    echo -e "${GREEN}✅ 正确返回404错误${NC}"
else
    echo -e "${YELLOW}⚠️  返回状态码: $http_code${NC}"
fi

echo -e "\n${GREEN}🎉 API测试完成!${NC}"
echo -e "\n${BLUE}提示:${NC}"
echo "- 使用 'python3 demo_api.py' 进行更详细的测试"
echo "- 使用 'python3 demo_api.py' 进入交互模式" 