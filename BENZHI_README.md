# BENZHI_README

基于 Go 实现的古籍数字化识别质量放行服务 HTTP API 项目，一款后端服务，已完整实现古籍数字化识别质量放行服务，贯通批次建档、证据登记、OCR 质检、问题整改、专家复核、证据冻结、凭据签发和独立验真流程，并通过规定的测试与自检命令。

## 项目说明
- 项目：benzhi-project-2e4c3bc9-22d0-4118-a3c8-7c134c1c4647
- 项目用途：已完整实现古籍数字化识别质量放行服务，贯通批次建档、证据登记、OCR 质检、问题整改、专家复核、证据冻结、凭据签发和独立验真流程，并通过规定的测试与自检命令。
- Go 工具链：`golang:1.26`
- 前端工具链：无

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run . -addr=127.0.0.1:19081 -selfcheck
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-2e4c3bc9-22d0-4118-a3c8-7c134c1c4647-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-2e4c3bc9-22d0-4118-a3c8-7c134c1c4647-arm64 linux/arm64
docker run -it benzhi-project-2e4c3bc9-22d0-4118-a3c8-7c134c1c4647-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run . -addr=127.0.0.1:19081 -selfcheck`
