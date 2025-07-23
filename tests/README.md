# 测试目录结构说明

## 概述

本项目采用统一的测试目录结构，将所有测试代码集中管理，便于维护和扩展。

## 目录结构

```
tests/
├── README.md                    # 测试说明文档
├── unit/                        # 单元测试
│   ├── internal/                # internal包的单元测试
│   │   ├── gpio/               # GPIO相关测试
│   │   ├── models/             # 数据模型测试
│   │   ├── services/           # 服务层测试
│   │   └── handlers/           # 处理器测试
│   ├── cmd/                    # cmd包的单元测试
│   │   ├── client/             # 客户端测试
│   │   └── server/             # 服务端测试
│   └── utils/                  # 工具函数测试
├── integration/                 # 集成测试
│   ├── api/                    # API集成测试
│   ├── websocket/              # WebSocket集成测试
│   └── gpio/                   # GPIO集成测试
├── e2e/                        # 端到端测试
│   ├── robot_control/          # 机器人控制测试
│   ├── game_flow/              # 游戏流程测试
│   └── system/                 # 系统级测试
├── fixtures/                   # 测试数据
│   ├── configs/                # 配置文件
│   ├── data/                   # 测试数据
│   └── mocks/                  # Mock数据
├── scripts/                    # 测试脚本
│   ├── run_tests.sh            # 运行测试脚本
│   ├── setup_test_env.sh       # 测试环境设置
│   └── cleanup.sh              # 清理脚本
└── reports/                    # 测试报告
    ├── coverage/               # 覆盖率报告
    ├── junit/                  # JUnit格式报告
    └── html/                   # HTML格式报告
```

## 测试分类

### 1. 单元测试 (Unit Tests)
- **位置**: `tests/unit/`
- **目的**: 测试单个函数、方法或类的功能
- **特点**: 
  - 快速执行
  - 独立性强
  - 不依赖外部资源
  - 高覆盖率

### 2. 集成测试 (Integration Tests)
- **位置**: `tests/integration/`
- **目的**: 测试模块间的交互
- **特点**:
  - 测试组件间协作
  - 可能依赖数据库、网络等
  - 执行时间中等

### 3. 端到端测试 (E2E Tests)
- **位置**: `tests/e2e/`
- **目的**: 测试完整的用户流程
- **特点**:
  - 模拟真实用户操作
  - 测试完整系统
  - 执行时间较长

## 命名规范

### 文件命名
- 单元测试: `*_test.go`
- 集成测试: `*_integration_test.go`
- 端到端测试: `*_e2e_test.go`

### 函数命名
- 单元测试: `TestFunctionName`
- 集成测试: `TestIntegration_FeatureName`
- 端到端测试: `TestE2E_UserFlow`

### 包命名
- 测试包名与源包名保持一致
- 使用 `package main` 或 `package packagename`

## 运行测试

### 运行所有测试
```bash
./tests/scripts/run_tests.sh
```

### 运行特定类型测试
```bash
# 单元测试
go test ./tests/unit/...

# 集成测试
go test ./tests/integration/...

# 端到端测试
go test ./tests/e2e/...
```

### 运行特定模块测试
```bash
# GPIO测试
go test ./tests/unit/internal/gpio/...

# 客户端测试
go test ./tests/unit/cmd/client/...

# 服务端测试
go test ./tests/unit/cmd/server/...
```

## 测试覆盖率

### 生成覆盖率报告
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o tests/reports/coverage/coverage.html
```

### 覆盖率目标
- 单元测试: ≥ 80%
- 集成测试: ≥ 60%
- 整体覆盖率: ≥ 70%

## 测试数据管理

### 测试配置文件
- 位置: `tests/fixtures/configs/`
- 格式: YAML/JSON
- 用途: 提供测试环境配置

### Mock数据
- 位置: `tests/fixtures/mocks/`
- 格式: JSON/YAML
- 用途: 模拟外部服务响应

### 测试数据
- 位置: `tests/fixtures/data/`
- 格式: 根据测试需要
- 用途: 提供测试输入数据

## 最佳实践

### 1. 测试隔离
- 每个测试应该独立运行
- 避免测试间的依赖关系
- 使用测试夹具和清理函数

### 2. 测试可读性
- 使用描述性的测试名称
- 遵循AAA模式 (Arrange, Act, Assert)
- 添加必要的注释

### 3. 测试维护性
- 避免重复代码
- 使用测试辅助函数
- 保持测试代码的简洁性

### 4. 性能考虑
- 单元测试应该快速执行
- 合理使用并行测试
- 避免不必要的资源消耗

## 持续集成

### CI/CD集成
- 自动运行测试套件
- 生成测试报告
- 检查覆盖率阈值
- 失败时阻止部署

### 测试环境
- 使用独立的测试环境
- 避免影响生产数据
- 提供测试数据库和配置 