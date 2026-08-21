# wje-132 建筑施工安全管理平台 执行记录

- 项目编号/名称：wje-132 建筑施工安全管理平台（SafetyPlatform）
- 日期：2026-08-16
- 短名：safety-platform
- 端口：前端 18702 / 后端 19202 / MySQL 33307（本机 3306 被 OrbStack 占用，.env 覆盖 DB_PORT=33307；compose 默认 3306）
- 技术栈：React 18 + TypeScript + Ant Design 5 + ECharts + Zustand + Vite；Go 1.22 + Gin + GORM；MySQL 8.0；JWT + RBAC

## Docker Compose 结果

| 容器 | 状态 | 端口 |
| --- | --- | --- |
| safety-platform-db | Up (healthy) | 33307:3306 |
| safety-platform-backend | Up (healthy) | 19202:8080 |
| safety-platform-frontend | Up | 18702:80 |

`docker compose config --quiet` 通过；`docker compose up -d --build` 一键启动成功。

## 关键 API 冒烟结果（36/36 通过）

| 接口 | 方法 | 状态码 | 结果摘要 |
| --- | --- | --- | --- |
| /healthz | GET | 200 | ok |
| /api/healthz（Nginx 反代） | GET | 200 | ok |
| /api/v1/healthz（Nginx 反代） | GET | 200 | ok |
| /api/v1/auth/register | POST | 200 | 注册成功 |
| /api/v1/auth/login（安全管理员） | POST | 200 | 返回 JWT + 用户 |
| /api/v1/users/me | GET | 200 | 当前用户 |
| /api/v1/users/me（未授权） | GET | 401 | 未登录 |
| /api/v1/users（工人） | GET | 403 | 越权拦截 |
| /api/v1/dashboard/stats | GET | 200 | 趋势/分布/待整改/完成率 |
| /api/v1/incidents?page_size=5 | GET | 200 | 事件列表 |
| /api/v1/incidents | POST | 200 | 上报事件 |
| /api/v1/incidents/:id | GET | 200 | 事件详情 |
| /api/v1/incidents/:id/assign | POST | 200 | 指派调查 |
| /api/v1/incidents/:id/rectify | POST | 200 | 提交整改 |
| /api/v1/incidents/:id/close | POST | 200 | 关闭事件 |
| /api/v1/incidents/:id/close（重复） | POST | 409 | 状态冲突 |
| /api/v1/incidents/:id/assign（工人） | POST | 403 | 越权拦截 |
| /api/v1/inspections?page_size=5 | GET | 200 | 检查计划列表 |
| /api/v1/inspections | POST | 200 | 创建检查计划 |
| /api/v1/inspections/:id | GET | 200 | 检查详情+检查项 |
| /api/v1/inspections/:id/execute | POST | 200 | 执行检查 |
| /api/v1/inspections/:id/report | GET | 200 | 检查报告 |
| /api/v1/trainings?page_size=5 | GET | 200 | 培训列表 |
| /api/v1/trainings | POST | 200 | 创建培训 |
| /api/v1/trainings/:id/record | POST | 200 | 记录成绩 |
| /api/v1/certifications?page_size=5 | GET | 200 | 资质列表 |
| /api/v1/certifications | POST | 200 | 提交资质 |
| /api/v1/certifications/:id/review | POST | 200 | 审核资质 |
| /api/v1/certifications/:id/review（重复） | POST | 409 | 状态冲突 |
| /api/v1/certifications?expiring=true | GET | 200 | 过期预警 |
| /api/v1/certifications/by-user | GET | 200 | 用户资质 |
| /api/v1/auth/login（admin） | POST | 200 | 管理员登录 |
| /api/v1/audit-logs（admin） | GET | 200 | 审计日志 |
| /api/v1/audit-logs（安全员） | GET | 403 | 越权拦截 |
| /api/v1/upload/image | POST | 200 | 图片上传返回 url |

异常路径覆盖：401 未授权、403 越权（用户列表/事件指派/审计日志）、409 状态冲突（事件关闭/资质审核）。

## 浏览器验证结论（内置 Playwright，无外部 Chrome）

- /login 登录页打开正常，输入 13800000002/User@123 登录后进入仪表盘，头部显示「王安全」与全部菜单（安全概览/事件管理/检查管理/培训管理/资质审核/审计日志/个人中心）。
- /dashboard 渲染统计卡片（检查计划总数/检查完成率/本月培训完成率/即将过期资质）与「待整改事件」表格，趋势图/饼图区域正常。
- /incidents 渲染真实事件：脚手架扣件松动（较大/调查中）、临时用电电缆破损（一般/已整改）、高处坠物未遂（轻微/已关闭）。
- /certifications 渲染真实资质（特种作业证/安全员证/电工证）与「过期预警」按钮。
- 截图：output/wje132_dashboard.png、output/wje132_incidents.png。

## README 检查项

- Docker Compose 一键启动命令在最前；本地开发命令；技术栈表格（后端 Go 1.22 + Gin + GORM）；目录结构；环境变量；部署说明；License。
- 枚举出现位置清单：SeverityLevel、IncidentStatus、UserRole 前后端出现位置已列出。

## 其他质量项

- 后端 `go build ./...` 通过；`go vet` 通过。
- 单元测试：internal/service 与 internal/util 表驱动测试通过（go test ./... ok）。
- 前端 `npm run build` 零错误（tsc + vite build 通过）。
- database/init.sql 含建表与种子数据（4 用户 + 3 事件 + 2 检查 + 3 检查项 + 2 培训 + 3 资质 + 审计日志），容器首次启动自动执行。

- 提交记录：init commit（见 git log）；本文件独立提交 docs commit。
