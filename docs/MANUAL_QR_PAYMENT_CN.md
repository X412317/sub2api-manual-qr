# 人工二维码分档充值运维指南

本文适用于 `v0.1.170` 的定制镜像 `0.1.170-manualpay.1`。该功能使用管理员上传的个人支付宝或微信收款码，用户提交付款截图和交易号后由管理员人工审核。它不监听收款通知，也不模拟或伪造支付回调。

## 功能边界

- 仅支持余额充值，不支持订阅购买。
- 固定档位为 `10、20、50、100、200、500、1000 CNY`。
- 必须保持 `1 CNY = 1 USD`，充值手续费必须为 `0%`。
- 仅支持支付宝和微信二维码；自动退款关闭，少付、多付和退款在线下处理。
- 用户提交凭证后订单不会自动过期且不能取消。驳回后重新获得一个完整订单超时时间。
- 每个订单最多提交 3 次；同一支付渠道的交易号全局不可重复。
- 批准入账必须通过管理员 TOTP 二次验证，并输入与订单实付金额完全一致的实收金额。

## 构建镜像

在仓库根目录执行：

```bash
./deploy/build_image.sh
```

脚本默认构建并写入版本号 `0.1.170-manualpay.1`。如需推送到私有仓库，可通过 `SUB2API_IMAGE` 覆盖镜像名；`SUB2API_VERSION` 仅用于覆盖容器内显示的构建版本：

```bash
SUB2API_IMAGE=registry.example.com/sub2api:0.1.170-manualpay.1 \
SUB2API_VERSION=0.1.170-manualpay.1 \
./deploy/build_image.sh
```

`deploy/.env` 中设置：

```dotenv
SUB2API_IMAGE=sub2api:0.1.170-manualpay.1
```

Compose 已配置独立私有目录：

```yaml
environment:
  - PAYMENT_PRIVATE_STORAGE_DIR=/data/payment-private
volumes:
  - ./data/payment-private:/data/payment-private
```

该目录不得由 Caddy、Nginx 或其他静态文件服务公开。容器中的图片接口均要求用户或管理员鉴权。

## 上线前备份

先停止写入或安排维护窗口，然后备份 PostgreSQL 和私有文件。以下命令在 `deploy` 目录执行，数据库容器名按实际 Compose 配置调整：

```bash
mkdir -p backups
docker compose exec -T postgres pg_dump -U "${POSTGRES_USER:-sub2api}" -d "${POSTGRES_DB:-sub2api}" -Fc > "backups/sub2api-before-manualpay.dump"
tar -czf "backups/payment-private-before-manualpay.tar.gz" data/payment-private 2>/dev/null || true
```

确认数据库备份不是空文件：

```bash
ls -lh backups/sub2api-before-manualpay.dump
```

## 首次配置

1. 使用新镜像启动服务。迁移 `194_manual_qr_payments.sql` 只新增表和索引，不删除原数据。
2. 先保持人工二维码服务商禁用。
3. 在“管理后台 -> 设置 -> 安全”启用 TOTP 和敏感操作二次验证，并为负责审核的管理员绑定 TOTP。
4. 在“管理后台 -> 设置 -> 支付”确认余额充值倍率为 `1`、手续费率为 `0`。
5. 启用支付宝和/或微信作为前台支付方式。
6. 新建“人工二维码”服务商，保存后重新编辑。
7. 分别上传支付宝、微信通用码；按需上传 7 个定额码。定额码优先，缺少时自动使用通用码。
8. 上传时服务端会重新解码图片、识别二维码载荷并剥离 EXIF。支付宝码与微信码不能混用。
9. 配置完成后再启用服务商。若同一前台支付方式已有官方或易支付实例，请先停用旧来源。

## 10 元验收

上线前必须使用普通测试账号完成一笔 `¥10`：

1. 打开 `/purchase`，选择 `10` 元和目标支付渠道。
2. 核对页面展示的金额、订单号、倒计时和二维码。
3. 实际付款 `¥10`，上传付款截图并填写真实交易号。
4. 确认提交后页面显示“待人工审核”，倒计时停止，用户不能取消订单。
5. 管理员打开“订单管理 -> 待审核”，预览私有截图并核对收款账户流水。
6. 在实收金额中输入 `10.00`，批准后完成 TOTP 验证。
7. 确认订单依次进入 `PAID`、`COMPLETED`，测试账号余额增加 `$10.00`，重复批准不会重复加款。
8. 另建测试订单验证驳回：填写原因后，用户应看到原因、完整的新倒计时和剩余提交次数。

## 日常审核规则

- 只以收款账户中的真实到账记录为准，不能只看用户截图。
- 交易号、渠道和实付金额必须一致；少付、多付或截图不清晰一律驳回。
- 管理员不能调整订单到账余额。补款或退款在线下完成后，再让用户按实际情况重新提交。
- 不要共享管理员账号或 TOTP。审核日志会记录审核人、结果、实收金额、原因和时间。
- 终态订单的截图保留 180 天后由后台任务清理；交易号、文件哈希和审核日志继续保留。

## 接口

用户接口：

- `POST /api/v1/payment/orders/:id/manual-proof`：`multipart/form-data`，字段为 `transaction_no` 和 `proof`。
- `GET /api/v1/payment/orders/:id/manual-proof`：读取本人订单最新凭证状态。
- `GET /api/v1/payment/orders/:id/manual-qr`：读取本人订单快照对应的私有二维码图片。

管理员接口：

- `GET/POST /api/v1/admin/payment/providers/:id/manual-qr-assets`
- `DELETE /api/v1/admin/payment/providers/:id/manual-qr-assets/:assetId`
- `GET /api/v1/admin/payment/providers/:id/manual-qr-assets/:assetId/image`
- `GET /api/v1/admin/payment/manual-reviews/summary`
- `GET /api/v1/admin/payment/orders/:id/manual-proof`
- `POST /api/v1/admin/payment/orders/:id/manual-review`

二维码只接受 PNG、JPEG、WebP 且不超过 1 MB；付款凭证不超过 5 MB。不要绕过接口直接读取存储目录。

## 无损回滚

1. 先在管理后台禁用所有“人工二维码”服务商，避免产生新订单。
2. 等待现有待审核订单处理完毕；如必须立即回滚，先记录未处理订单并暂停充值入口。
3. 将 `SUB2API_IMAGE` 切回回滚前镜像并执行 `docker compose up -d`。
4. 不删除 `manual_payment_qr_assets`、`manual_payment_proofs`、审核日志或 `data/payment-private`。

旧镜像不会使用新增表，保留它们不会影响旧功能。重新切回定制镜像后，旧订单仍使用下单时固化的二维码快照。
